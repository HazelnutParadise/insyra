// fa/lapack_dsyevr.go
//
// dsyevr — public driver for symmetric eigenproblem.
//
//   1. dsytrd: A → tridiagonal T = Q^T A Q
//   2. dstemr (MRRR) for eigenvalues + eigenvectors of T
//      (fallback to dstebz/dstein if MRRR fails)
//   3. dormtr: apply Q to map eigenvectors back to A's basis
//   4. rescale + sort
//
// Faithful translation of LAPACK 3.12.1 dsyevr.f. Currently restricted
// to UPLO='L' (matches dormtr's only-implemented path).
package fa

import "math"

// dsyevr computes selected eigenvalues and (optionally) eigenvectors
// of a real symmetric matrix.
//
//	jobz   : 'N' (eigenvalues only) or 'V' (eigenvalues + eigenvectors)
//	rang   : 'A' (all), 'V' (interval), 'I' (indices)
//	uplo   : 'L' only currently supported (panics on 'U' via dormtr)
//	n      : matrix order
//	a      : column-major flat lda x n; modified in place by dsytrd
//	lda    : leading dimension of a
//	vl, vu : value range (when rang='V')
//	il, iu : index range (when rang='I')
//	abstol : absolute tolerance (use ulp*||T|| if <= 0)
//	z      : column-major flat ldz x m output (eigenvectors as columns)
//	ldz    : leading dimension of z
//	work   : workspace, length 26*n
//	iwork  : workspace, length 10*n
//
// Outputs:
//
//	m      : number of eigenvalues found
//	w      : eigenvalues (length n; first m valid)
//	isuppz : eigenvector support, length 2*m
//	info   : 0 ok; <0 illegal arg; >0 internal failure
func dsyevr(
	jobz, rang, uplo byte,
	n int,
	a []float64,
	lda int,
	vl, vu float64,
	il, iu int,
	abstol float64,
	w []float64,
	z []float64,
	ldz int,
	isuppz []int,
	work []float64,
	iwork []int,
) (m, info int) {
	const (
		eps    = 2.2204460492503131e-16
		safmin = lapackSafmin
	)
	wantz := jobz == 'V' || jobz == 'v'
	alleig := rang == 'A' || rang == 'a'
	valeig := rang == 'V' || rang == 'v'
	indeig := rang == 'I' || rang == 'i'
	lower := uplo == 'L' || uplo == 'l'
	upper := uplo == 'U' || uplo == 'u'

	if !(wantz || jobz == 'N' || jobz == 'n') {
		return 0, -1
	}
	if !(alleig || valeig || indeig) {
		return 0, -2
	}
	if !(lower || upper) {
		return 0, -3
	}
	if n < 0 {
		return 0, -4
	}
	if lda < max(1, n) {
		return 0, -6
	}
	if valeig && n > 0 && vu <= vl {
		return 0, -8
	}
	if indeig && (il < 1 || il > max(1, n)) {
		return 0, -9
	}
	if indeig && (iu < min(n, il) || iu > n) {
		return 0, -10
	}
	if ldz < 1 || (wantz && ldz < n) {
		return 0, -15
	}

	if n == 0 {
		return 0, 0
	}
	if n == 1 {
		if alleig || indeig {
			m = 1
			w[0] = a[0]
		} else {
			if vl < a[0] && vu >= a[0] {
				m = 1
				w[0] = a[0]
			}
		}
		if wantz {
			z[0] = 1
			isuppz[0] = 1
			isuppz[1] = 1
		}
		return m, 0
	}

	smlnum := safmin / eps
	bignum := 1.0 / smlnum
	rmin := math.Sqrt(smlnum)
	rmax := math.Min(math.Sqrt(bignum), 1.0/math.Sqrt(math.Sqrt(safmin)))

	// Scale A if needed.
	iscale := 0
	abstll := abstol
	vll := vl
	vuu := vu
	anrm := dlansy('M', uplo, n, a, lda, work)
	var sigma float64
	if anrm > 0 && anrm < rmin {
		iscale = 1
		sigma = rmin / anrm
	} else if anrm > rmax {
		iscale = 1
		sigma = rmax / anrm
	}
	if iscale == 1 {
		if lower {
			for j := 1; j <= n; j++ {
				dscal(n-j+1, sigma, a[(j-1)*lda+j-1:], 1)
			}
		} else {
			for j := 1; j <= n; j++ {
				dscal(j, sigma, a[(j-1)*lda:], 1)
			}
		}
		if abstol > 0 {
			abstll = abstol * sigma
		}
		if valeig {
			vll = vl * sigma
			vuu = vu * sigma
		}
	}

	// Workspace partition.
	indtau := 0
	indd := indtau + n
	inde := indd + n
	inddd := inde + n
	indee := inddd + n
	indwk := indee + n

	indibl := 0
	indisp := indibl + n
	indifl := indisp + n
	indiwo := indifl + n

	// dsytrd: A → tridiagonal.
	dsytrd(uplo, n, a, lda, work[indd:], work[inde:], work[indtau:])

	// Try dstemr (MRRR) for ALLEIG / full-INDEIG.
	tryStemr := alleig || (indeig && il == 1 && iu == n)
	stemrSucceeded := false
	if tryStemr {
		if !wantz {
			// JOBZ='N': dsterf on a copy.
			copy(w[:n], work[indd:indd+n])
			copy(work[indee:indee+n-1], work[inde:inde+n-1])
			info = dsterf(n, w, work[indee:])
		} else {
			copy(work[indee:indee+n-1], work[inde:inde+n-1])
			copy(work[inddd:inddd+n], work[indd:indd+n])
			tryrac := abstol <= 2*float64(n)*eps
			var stemrInfo int
			m, stemrInfo = dstemr(jobz, 'A', n,
				work[inddd:], work[indee:],
				vl, vu, il, iu,
				w, z, ldz, n,
				isuppz, tryrac,
				work[indwk:], iwork)
			if stemrInfo != 0 {
				info = stemrInfo
			} else {
				// Apply Q via dormtr.
				indwkn := inde
				dormtr('L', uplo, 'N', n, m, a, lda,
					work[indtau:], z, ldz,
					work[indwkn:])
				info = 0
				stemrSucceeded = true
			}
		}
		if info == 0 {
			if !wantz {
				m = n
			}
			goto rescale
		}
		// dstemr / dsterf failed; fall through.
		info = 0
	}

	// dstebz + dstein fallback.
	{
		var orderC byte
		if wantz {
			orderC = 'B'
		} else {
			orderC = 'E'
		}
		var nsplit int
		var stebzInfo int
		m, nsplit, stebzInfo = dstebz(rang, orderC, n, vll, vuu, il, iu, abstll,
			work[indd:], work[inde:],
			w, iwork[indibl:], iwork[indisp:],
			work[indwk:], iwork[indiwo:])
		if stebzInfo != 0 {
			info = stebzInfo
			goto rescale
		}
		_ = nsplit

		if wantz {
			steinInfo := dstein(n, work[indd:], work[inde:], m, w,
				iwork[indibl:], iwork[indisp:],
				z, ldz,
				work[indwk:], iwork[indiwo:], iwork[indifl:])
			if steinInfo != 0 {
				info = steinInfo
				goto rescale
			}
			// Apply Q.
			indwkn := inde
			dormtr('L', uplo, 'N', n, m, a, lda,
				work[indtau:], z, ldz,
				work[indwkn:])
		}
	}

rescale:
	if iscale == 1 {
		imax := m
		if info != 0 {
			imax = info - 1
			if imax < 0 {
				imax = 0
			}
		}
		dscal(imax, 1.0/sigma, w, 1)
	}

	// Sort if needed (always for the dstebz/dstein path; dstemr already sorted).
	if wantz && !stemrSucceeded {
		for j := 1; j <= m-1; j++ {
			i := 0
			tmp1 := w[j-1]
			for jj := j + 1; jj <= m; jj++ {
				if w[jj-1] < tmp1 {
					i = jj
					tmp1 = w[jj-1]
				}
			}
			if i != 0 {
				w[i-1] = w[j-1]
				w[j-1] = tmp1
				dswap(n, z[(i-1)*ldz:(i-1)*ldz+n], 1, z[(j-1)*ldz:(j-1)*ldz+n], 1)
			}
		}
	}
	return m, info
}
