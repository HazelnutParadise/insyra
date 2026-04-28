// fa/lapack_dstemr.go
//
// dstemr — compute selected eigenvalues and (optionally) eigenvectors
// of a real symmetric tridiagonal matrix using the MRRR algorithm
// (multiple-relatively-robust-representations). Driver that calls
// dlarre + dlarrv. Faithful translation of LAPACK 3.12.1 dstemr.f.
package fa

import "math"

// dstemr top-level signature.
//
//	jobz    : 'N' (eigenvalues only) or 'V' (eigenvalues + eigenvectors)
//	rang    : 'A' (all), 'V' (interval), 'I' (indices)
//	n       : matrix order
//	d       : diagonal (length n; OVERWRITTEN by scaling/factorization)
//	e       : off-diagonal (length n; e[n-1] arbitrary; OVERWRITTEN)
//	vl, vu  : value range (used when rang='V')
//	il, iu  : index range (used when rang='I')
//	z       : column-major flat ldz x m output (eigenvectors as columns)
//	ldz     : leading dimension of z (>= n if wantz)
//	nzc     : number of cols allocated in z (must be >= m or -1 for query)
//	tryrac  : whether to attempt high relative accuracy
//	work    : workspace, length 18*n (or 12*n if !wantz)
//	iwork   : workspace, length 10*n (or 8*n if !wantz)
//
// Outputs:
//
//	m       : number of eigenvalues found
//	w       : eigenvalues (length n; first m valid)
//	isuppz  : eigenvector support, length 2*m
//	info    : 0 ok; positive = various failure modes per reference
func dstemr(
	jobz, rang byte,
	n int,
	d, e []float64,
	vl, vu float64,
	il, iu int,
	w []float64,
	z []float64,
	ldz, nzc int,
	isuppz []int,
	tryrac bool,
	work []float64,
	iwork []int,
) (m int, info int) {
	const (
		minrgp = 1e-3
		eps    = 2.2204460492503131e-16
		safmin = lapackSafmin
	)

	wantz := jobz == 'V' || jobz == 'v'
	alleig := rang == 'A' || rang == 'a'
	valeig := rang == 'V' || rang == 'v'
	indeig := rang == 'I' || rang == 'i'

	if !(wantz || jobz == 'N' || jobz == 'n') {
		return 0, -1
	}
	if !(alleig || valeig || indeig) {
		return 0, -2
	}
	if n < 0 {
		return 0, -3
	}
	wl := 0.0
	wu := 0.0
	iil := 0
	iiu := 0
	if valeig {
		wl = vl
		wu = vu
		if n > 0 && wu <= wl {
			return 0, -7
		}
	} else if indeig {
		iil = il
		iiu = iu
		if iil < 1 || iil > n {
			return 0, -8
		}
		if iiu < iil || iiu > n {
			return 0, -9
		}
	}
	if ldz < 1 || (wantz && ldz < n) {
		return 0, -13
	}

	smlnum := safmin / eps
	bignum := 1.0 / smlnum
	rmin := math.Sqrt(smlnum)
	rmax := math.Min(math.Sqrt(bignum), 1.0/math.Sqrt(math.Sqrt(safmin)))

	if n == 0 {
		return 0, 0
	}
	if n == 1 {
		if alleig || indeig {
			m = 1
			w[0] = d[0]
		} else {
			if wl < d[0] && wu >= d[0] {
				m = 1
				w[0] = d[0]
			}
		}
		if wantz {
			z[0] = 1
			isuppz[0] = 1
			isuppz[1] = 1
		}
		_ = nzc
		return m, 0
	}

	if n == 2 {
		var r1, r2, cs, sn float64
		laeswap := false
		if !wantz {
			r1, r2 = dlae2(d[0], e[0], d[1])
		} else {
			r1, r2, cs, sn = dlaev2(d[0], e[0], d[1])
		}
		if r1 < r2 {
			r1, r2 = r2, r1
			laeswap = true
		}
		take2 := func(rv float64, idx int) {
			if alleig ||
				(valeig && rv > wl && rv <= wu) ||
				(indeig && ((idx == 1 && iil == 1) || (idx == 2 && iiu == 2))) {
				m++
				w[m-1] = rv
				if wantz {
					col := (m - 1) * ldz
					if idx == 1 {
						// r2 (smaller): use (-sn, cs) or (cs, sn) per laeswap
						if laeswap {
							z[col+0] = cs
							z[col+1] = sn
						} else {
							z[col+0] = -sn
							z[col+1] = cs
						}
					} else {
						// r1 (larger)
						if laeswap {
							z[col+0] = -sn
							z[col+1] = cs
						} else {
							z[col+0] = cs
							z[col+1] = sn
						}
					}
					if sn != 0 {
						if cs != 0 {
							isuppz[2*(m-1)] = 1
							isuppz[2*(m-1)+1] = 2
						} else {
							isuppz[2*(m-1)] = 1
							isuppz[2*(m-1)+1] = 1
						}
					} else {
						isuppz[2*(m-1)] = 2
						isuppz[2*(m-1)+1] = 2
					}
				}
			}
		}
		take2(r2, 1)
		take2(r1, 2)
		// Sort if needed (always check both wantz and non-wantz paths).
		if m >= 2 && wantz {
			// 2x2 ascending check.
			if w[0] > w[1] {
				w[0], w[1] = w[1], w[0]
				dswap(n, z[0:n], 1, z[ldz:ldz+n], 1)
				isuppz[0], isuppz[2] = isuppz[2], isuppz[0]
				isuppz[1], isuppz[3] = isuppz[3], isuppz[1]
			}
		}
		return m, 0
	}

	// General N >= 3.
	indgrs := 0
	inderr := 2 * n
	indgp := 3 * n
	indd := 4 * n
	inde2 := 5 * n
	indwrk := 6 * n
	iinspl := 0
	iindbl := n
	iindw := 2 * n
	iindwk := 3 * n

	// Scale to allowable range if necessary.
	scale := 1.0
	tnrm := dlanst('M', n, d, e)
	if tnrm > 0 && tnrm < rmin {
		scale = rmin / tnrm
	} else if tnrm > rmax {
		scale = rmax / tnrm
	}
	if scale != 1 {
		dscal(n, scale, d, 1)
		dscal(n-1, scale, e, 1)
		tnrm *= scale
		if valeig {
			wl *= scale
			wu *= scale
		}
	}

	// Splitting criterion.
	var thresh float64
	if tryrac {
		iinfo := dlarrr(n, d, e)
		if iinfo == 0 {
			thresh = eps
		} else {
			thresh = -eps
			tryrac = false
		}
	} else {
		thresh = -eps
	}
	if tryrac {
		copy(work[indd:indd+n], d[:n])
	}
	for j := 1; j <= n-1; j++ {
		work[inde2+j-1] = e[j-1] * e[j-1]
	}

	var rtol1, rtol2 float64
	if !wantz {
		rtol1 = 4 * eps
		rtol2 = 4 * eps
	} else {
		rtol1 = math.Sqrt(eps)
		rtol2 = math.Max(math.Sqrt(eps)*5e-3, 4*eps)
	}

	var pivmin float64
	var nsplit, mloc, iinfoLarre int
	wl, wu, mloc, nsplit, pivmin, iinfoLarre = dlarre(rang, n, wl, wu, iil, iiu,
		d, e, work[inde2:],
		rtol1, rtol2, thresh,
		iwork[iinspl:],
		w, work[inderr:], work[indgp:],
		iwork[iindbl:], iwork[iindw:],
		work[indgrs:],
		work[indwrk:], iwork[iindwk:])
	if iinfoLarre != 0 {
		return 0, 10 + abs(iinfoLarre)
	}
	m = mloc

	if wantz {
		iinfoLarrv := dlarrv(n, wl, wu, d, e, pivmin,
			iwork[iinspl:], m, 1, m, minrgp, rtol1, rtol2,
			w, work[inderr:], work[indgp:],
			iwork[iindbl:], iwork[iindw:],
			work[indgrs:],
			z, ldz, isuppz,
			work[indwrk:], iwork[iindwk:])
		if iinfoLarrv != 0 {
			return m, 20 + abs(iinfoLarrv)
		}
	} else {
		// Apply per-block shifts to the eigenvalues.
		for j := 1; j <= m; j++ {
			itmp := iwork[iindbl+j-1]
			w[j-1] = w[j-1] + e[iwork[iinspl+itmp-1]-1]
		}
	}

	if tryrac {
		ibegin := 1
		wbegin := 1
		for jblk := 1; jblk <= iwork[iindbl+m-1]; jblk++ {
			iend := iwork[iinspl+jblk-1]
			in := iend - ibegin + 1
			wend := wbegin - 1
			for wend < m && iwork[iindbl+wend] == jblk {
				wend++
			}
			if wend < wbegin {
				ibegin = iend + 1
				continue
			}
			offset := iwork[iindw+wbegin-1] - 1
			ifirst := iwork[iindw+wbegin-1]
			ilast := iwork[iindw+wend-1]
			rtol2 = 4 * eps
			dlarrj(in, work[indd+ibegin-1:], work[inde2+ibegin-1:],
				ifirst, ilast, rtol2, offset,
				w[wbegin-1:], work[inderr+wbegin-1:],
				work[indwrk:], iwork[iindwk:], pivmin, tnrm)
			ibegin = iend + 1
			wbegin = wend + 1
		}
	}

	if scale != 1 {
		dscal(m, 1.0/scale, w, 1)
	}

	// Sort eigenvalues (and matching eigenvectors) into ascending order.
	if nsplit > 1 || n == 2 {
		if !wantz {
			dlasrt('I', m, w)
		} else {
			for j := 1; j <= m-1; j++ {
				ie := 0
				tmp := w[j-1]
				for jj := j + 1; jj <= m; jj++ {
					if w[jj-1] < tmp {
						ie = jj
						tmp = w[jj-1]
					}
				}
				if ie != 0 {
					w[ie-1] = w[j-1]
					w[j-1] = tmp
					dswap(n, z[(ie-1)*ldz:(ie-1)*ldz+n], 1, z[(j-1)*ldz:(j-1)*ldz+n], 1)
					itmp := isuppz[2*(ie-1)]
					isuppz[2*(ie-1)] = isuppz[2*(j-1)]
					isuppz[2*(j-1)] = itmp
					itmp = isuppz[2*(ie-1)+1]
					isuppz[2*(ie-1)+1] = isuppz[2*(j-1)+1]
					isuppz[2*(j-1)+1] = itmp
				}
			}
		}
	}

	_ = nzc
	return m, 0
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
