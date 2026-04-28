// fa/lapack_dsyev.go
//
// Symmetric eigenvalue solver path: dsteqr (QL/QR iteration on a
// symmetric tridiagonal matrix) plus the dsyev top-level driver that
// chains dsytrd → dorgtr → dsteqr to compute all eigenvalues and
// optionally eigenvectors of a symmetric matrix. Faithful translations
// of LAPACK 3.12.1 reference Fortran (dsteqr.f, dsyev.f, dlae2.f,
// dlaset.f, dlascl.f).
//
// This is the older (pre-MRRR) eigenvalue path. Gonum uses an equivalent
// dsyev internally, so this primarily serves as a *diagnostic* port —
// if our dsyev gives identical results to gonum's, the source of the R
// parity gap must be the dsyev↔dsyevr algorithm difference (R uses the
// MRRR dsyevr), and we have to push on with the full dsyevr port.
package fa

import "math"

// dlae2 computes the eigenvalues of a 2×2 symmetric matrix
// [[a, b], [b, c]] with rt1 the larger and rt2 the smaller in absolute
// value. Mirrors LAPACK dlae2.f.
func dlae2(a, b, c float64) (rt1, rt2 float64) {
	sm := a + c
	df := a - c
	adf := math.Abs(df)
	tb := b + b
	ab := math.Abs(tb)
	var acmx, acmn float64
	if math.Abs(a) > math.Abs(c) {
		acmx, acmn = a, c
	} else {
		acmx, acmn = c, a
	}
	var rt float64
	switch {
	case adf > ab:
		rt = adf * math.Sqrt(1+(ab/adf)*(ab/adf))
	case adf < ab:
		rt = ab * math.Sqrt(1+(adf/ab)*(adf/ab))
	default:
		rt = ab * math.Sqrt(2)
	}
	switch {
	case sm < 0:
		rt1 = 0.5 * (sm - rt)
		rt2 = (acmx/rt1)*acmn - (b/rt1)*b
	case sm > 0:
		rt1 = 0.5 * (sm + rt)
		rt2 = (acmx/rt1)*acmn - (b/rt1)*b
	default:
		rt1 = 0.5 * rt
		rt2 = -0.5 * rt
	}
	return
}

// dlaset initializes an m×n matrix A with diagonal `diag` and off-diagonal
// `offd`. uplo='U' upper, 'L' lower, 'F' full. Mirrors LAPACK dlaset.f.
func dlaset(uplo byte, m, n int, offd, diag float64, a []float64, lda int) {
	switch uplo {
	case 'U', 'u':
		for j := 1; j < n; j++ {
			imax := j
			if imax > m {
				imax = m
			}
			for i := 0; i < imax; i++ {
				a[j*lda+i] = offd
			}
		}
	case 'L', 'l':
		for j := 0; j < n; j++ {
			for i := j + 1; i < m; i++ {
				a[j*lda+i] = offd
			}
		}
	default:
		for j := 0; j < n; j++ {
			for i := 0; i < m; i++ {
				a[j*lda+i] = offd
			}
		}
	}
	mn := m
	if n < mn {
		mn = n
	}
	for i := 0; i < mn; i++ {
		a[i*lda+i] = diag
	}
}

// dlasclG implements LAPACK dlascl with TYPE='G' (general matrix scaling).
// Multiplies an m×n matrix A by cto/cfrom without overflow / underflow.
// Mirrors the loop in LAPACK dlascl.f's general case.
func dlasclG(cfrom, cto float64, m, n int, a []float64, lda int) {
	smlnum := lapackSafmin
	bignum := 1.0 / smlnum

	cfromc := cfrom
	ctoc := cto
	for done := false; !done; {
		cfrom1 := cfromc * smlnum
		var mul float64
		if cfrom1 == cfromc {
			// cfromc is an inf — no scaling needed
			mul = ctoc / cfromc
			done = true
		} else {
			cto1 := ctoc / bignum
			switch {
			case cto1 == ctoc:
				mul = ctoc
				done = true
				cfromc = 1
			case math.Abs(cfrom1) > math.Abs(ctoc) && ctoc != 0:
				mul = smlnum
				done = false
				cfromc = cfrom1
			case math.Abs(cto1) > math.Abs(cfromc):
				mul = bignum
				done = false
				ctoc = cto1
			default:
				mul = ctoc / cfromc
				done = true
			}
		}
		for j := 0; j < n; j++ {
			for i := 0; i < m; i++ {
				a[j*lda+i] *= mul
			}
		}
	}
}

// dsteqr computes all eigenvalues and (optionally) eigenvectors of a
// symmetric tridiagonal matrix using the implicit-shift QL/QR algorithm.
// Mirrors LAPACK dsteqr.f exactly.
//
// compz: 'N' eigenvalues only, 'V' eigenvectors of original matrix
//        (z is initial Q from dsytrd), 'I' z is set to identity.
// d: diagonal (length n), modified in-place to hold eigenvalues
// e: off-diagonal (length n-1), destroyed
// z: n×n matrix (lda=ldz), modified for compz != 'N'
// work: scratch length max(1, 2*n-2)
func dsteqr(compz byte, n int, d, e []float64, z []float64, ldz int, work []float64) (info int) {
	const maxIt = 30
	icompz := -1
	switch compz {
	case 'N', 'n':
		icompz = 0
	case 'V', 'v':
		icompz = 1
	case 'I', 'i':
		icompz = 2
	}
	if icompz < 0 {
		return -1
	}
	if n < 0 {
		return -2
	}
	if n == 0 {
		return 0
	}
	if n == 1 {
		if icompz == 2 {
			z[0] = 1
		}
		return 0
	}
	const eps = 2.2204460492503131e-16
	eps2 := eps * eps
	const safmin = lapackSafmin
	safmax := 1.0 / safmin
	ssfmax := math.Sqrt(safmax) / 3
	ssfmin := math.Sqrt(safmin) / eps2

	if icompz == 2 {
		dlaset('F', n, n, 0, 1, z, ldz)
	}

	nmaxit := n * maxIt
	jtot := 0

	l1 := 1
	nm1 := n - 1

	for {
		// label 10
		if l1 > n {
			break // → label 160 (sort)
		}
		if l1 > 1 {
			e[l1-2] = 0
		}
		// Find a small off-diagonal element to split.
		var m int
		if l1 <= nm1 {
			found := false
			for mm := l1; mm <= nm1; mm++ {
				tst := math.Abs(e[mm-1])
				if tst == 0 {
					m = mm
					found = true
					break
				}
				if tst <= math.Sqrt(math.Abs(d[mm-1]))*math.Sqrt(math.Abs(d[mm]))*eps {
					e[mm-1] = 0
					m = mm
					found = true
					break
				}
			}
			if !found {
				m = n
			}
		} else {
			m = n
		}

		l := l1
		lsv := l
		lend := m
		lendsv := lend
		l1 = m + 1
		if lend == l {
			continue
		}

		// Scale.
		anorm := dlanst('M', lend-l+1, d[l-1:], e[l-1:])
		iscale := 0
		if anorm == 0 {
			continue
		}
		if anorm > ssfmax {
			iscale = 1
			dlasclG(anorm, ssfmax, lend-l+1, 1, d[l-1:], n)
			if lend > l {
				dlasclG(anorm, ssfmax, lend-l, 1, e[l-1:], n)
			}
		} else if anorm < ssfmin {
			iscale = 2
			dlasclG(anorm, ssfmin, lend-l+1, 1, d[l-1:], n)
			if lend > l {
				dlasclG(anorm, ssfmin, lend-l, 1, e[l-1:], n)
			}
		}

		// Choose QL or QR.
		if math.Abs(d[lend-1]) < math.Abs(d[l-1]) {
			lend = lsv
			l = lendsv
		}

		if lend > l {
			// QL Iteration loop
		ql:
			for {
				// label 40
				if l != lend {
					lendm1 := lend - 1
					found := false
					for mm := l; mm <= lendm1; mm++ {
						tst := math.Abs(e[mm-1])
						tst = tst * tst
						if tst <= (eps2*math.Abs(d[mm-1]))*math.Abs(d[mm])+safmin {
							m = mm
							found = true
							break
						}
					}
					if !found {
						m = lend
					}
				} else {
					m = lend
				}
				if m < lend {
					e[m-1] = 0
				}
				p := d[l-1]
				if m == l {
					// label 80: eigenvalue found
					d[l-1] = p
					l++
					if l <= lend {
						continue ql
					}
					break ql
				}
				// 2x2 block
				if m == l+1 {
					var rt1, rt2, c, s float64
					if icompz > 0 {
						rt1, rt2, c, s = dlaev2(d[l-1], e[l-1], d[l])
						work[l-1] = c
						work[n-1+l-1] = s
						dlasr('R', 'V', 'B', n, 2, work[l-1:], work[n-1+l-1:],
							z[(l-1)*ldz:], ldz)
					} else {
						rt1, rt2 = dlae2(d[l-1], e[l-1], d[l])
					}
					d[l-1] = rt1
					d[l] = rt2
					e[l-1] = 0
					l += 2
					if l <= lend {
						continue ql
					}
					break ql
				}
				if jtot == nmaxit {
					break ql
				}
				jtot++
				// Form shift.
				g := (d[l] - p) / (2 * e[l-1])
				r := dlapy2(g, 1)
				g = d[m-1] - p + (e[l-1] / (g + math.Copysign(r, g)))

				s := 1.0
				c := 1.0
				p = 0.0
				mm1 := m - 1
				for i := mm1; i >= l; i-- {
					f := s * e[i-1]
					b := c * e[i-1]
					var rNew float64
					c, s, rNew = dlartg(g, f)
					if i != m-1 {
						e[i] = rNew
					}
					g = d[i] - p
					r = (d[i-1]-g)*s + 2*c*b
					p = s * r
					d[i] = g + p
					g = c*r - b
					if icompz > 0 {
						work[i-1] = c
						work[n-1+i-1] = -s
					}
				}
				if icompz > 0 {
					mm := m - l + 1
					dlasr('R', 'V', 'B', n, mm, work[l-1:], work[n-1+l-1:],
						z[(l-1)*ldz:], ldz)
				}
				d[l-1] -= p
				e[l-1] = g
			}
		} else {
			// QR Iteration loop
		qr:
			for {
				// label 90
				if l != lend {
					lendp1 := lend + 1
					found := false
					for mm := l; mm >= lendp1; mm-- {
						tst := math.Abs(e[mm-2])
						tst = tst * tst
						if tst <= (eps2*math.Abs(d[mm-1]))*math.Abs(d[mm-2])+safmin {
							m = mm
							found = true
							break
						}
					}
					if !found {
						m = lend
					}
				} else {
					m = lend
				}
				if m > lend {
					e[m-2] = 0
				}
				p := d[l-1]
				if m == l {
					// eigenvalue found
					d[l-1] = p
					l--
					if l >= lend {
						continue qr
					}
					break qr
				}
				// 2x2 block
				if m == l-1 {
					var rt1, rt2, c, s float64
					if icompz > 0 {
						rt1, rt2, c, s = dlaev2(d[l-2], e[l-2], d[l-1])
						work[m-1] = c
						work[n-1+m-1] = s
						dlasr('R', 'V', 'F', n, 2, work[m-1:], work[n-1+m-1:],
							z[(l-2)*ldz:], ldz)
					} else {
						rt1, rt2 = dlae2(d[l-2], e[l-2], d[l-1])
					}
					d[l-2] = rt1
					d[l-1] = rt2
					e[l-2] = 0
					l -= 2
					if l >= lend {
						continue qr
					}
					break qr
				}
				if jtot == nmaxit {
					break qr
				}
				jtot++
				// Form shift.
				g := (d[l-2] - p) / (2 * e[l-2])
				r := dlapy2(g, 1)
				g = d[m-1] - p + (e[l-2] / (g + math.Copysign(r, g)))

				s := 1.0
				c := 1.0
				p = 0.0
				lm1 := l - 1
				for i := m; i <= lm1; i++ {
					f := s * e[i-1]
					b := c * e[i-1]
					var rNew float64
					c, s, rNew = dlartg(g, f)
					if i != m {
						e[i-2] = rNew
					}
					g = d[i-1] - p
					r = (d[i]-g)*s + 2*c*b
					p = s * r
					d[i-1] = g + p
					g = c*r - b
					if icompz > 0 {
						work[i-1] = c
						work[n-1+i-1] = s
					}
				}
				if icompz > 0 {
					mm := l - m + 1
					dlasr('R', 'V', 'F', n, mm, work[m-1:], work[n-1+m-1:],
						z[(m-1)*ldz:], ldz)
				}
				d[l-1] -= p
				e[lm1-1] = g
			}
		}

		// Undo scaling.
		if iscale == 1 {
			dlasclG(ssfmax, anorm, lendsv-lsv+1, 1, d[lsv-1:], n)
			if lendsv > lsv {
				dlasclG(ssfmax, anorm, lendsv-lsv, 1, e[lsv-1:], n)
			}
		} else if iscale == 2 {
			dlasclG(ssfmin, anorm, lendsv-lsv+1, 1, d[lsv-1:], n)
			if lendsv > lsv {
				dlasclG(ssfmin, anorm, lendsv-lsv, 1, e[lsv-1:], n)
			}
		}

		if jtot >= nmaxit {
			for i := 0; i < n-1; i++ {
				if e[i] != 0 {
					info++
				}
			}
			return info
		}
	}

	// Sort eigenvalues (and reorder eigenvectors if computed).
	if icompz == 0 {
		dlasrt('I', n, d)
	} else {
		// Selection sort minimizing eigenvector swaps.
		for ii := 2; ii <= n; ii++ {
			i := ii - 1
			k := i
			p := d[i-1]
			for j := ii; j <= n; j++ {
				if d[j-1] < p {
					k = j
					p = d[j-1]
				}
			}
			if k != i {
				d[k-1] = d[i-1]
				d[i-1] = p
				dswap(n, z[(i-1)*ldz:], 1, z[(k-1)*ldz:], 1)
			}
		}
	}
	return info
}

// dsyev is the top-level eigen driver: A = Q diag(w) Q^T for symmetric A.
// jobz='N' eigenvalues only; 'V' compute eigenvectors. uplo='U'/'L'.
// Mirrors LAPACK dsyev.f (unblocked path).
func dsyev(jobz, uplo byte, n int, a []float64, lda int, w []float64) (info int) {
	if n == 0 {
		return 0
	}
	if n == 1 {
		w[0] = a[0]
		if jobz == 'V' || jobz == 'v' {
			a[0] = 1
		}
		return 0
	}

	wantz := jobz == 'V' || jobz == 'v'
	tau := make([]float64, n-1)
	e := make([]float64, n-1)

	// Reduce to tridiagonal form.
	dsytrd(uplo, n, a, lda, w, e, tau)

	// Compute eigenvalues / eigenvectors.
	if !wantz {
		// Eigenvalues only via QR/QL iteration on the tridiagonal.
		ee := make([]float64, n-1)
		copy(ee, e)
		work := make([]float64, max(1, 2*n-2))
		info = dsteqr('N', n, w, ee, nil, 1, work)
	} else {
		// Generate Q from the reflectors stored in A.
		dorgtr(uplo, n, a, lda, tau)
		work := make([]float64, max(1, 2*n-2))
		info = dsteqr('V', n, w, e, a, lda, work)
	}
	return info
}
