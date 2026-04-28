// fa/lapack_dsterf.go
//
// dsterf — eigenvalues-only QL/QR with implicit shift on a symmetric
// tridiagonal matrix. Auxiliary used by dsyevr's JOBZ='N' path.
// Faithful translation of LAPACK 3.12.1 dsterf.f.
package fa

import "math"

// dsterf computes all eigenvalues of a symmetric tridiagonal matrix
// (diagonal d, off-diagonal e) using the Pal-Walker-Kahan variant of
// the QL/QR algorithm. d is overwritten with the eigenvalues in
// ascending order; e is destroyed.
func dsterf(n int, d, e []float64) (info int) {
	if n < 0 {
		return -1
	}
	if n <= 1 {
		return 0
	}
	const (
		maxit  = 30
		eps    = 2.2204460492503131e-16
		safmin = lapackSafmin
	)
	eps2 := eps * eps
	safmax := 1.0 / safmin
	ssfmax := math.Sqrt(safmax) / 3
	ssfmin := math.Sqrt(safmin) / eps2

	nmaxit := n * maxit
	jtot := 0
	l1 := 1

	// Outer loop: locate next unreduced sub-block.
	for {
		if l1 > n {
			break
		}
		if l1 > 1 {
			e[l1-2] = 0
		}
		mm := n
		for mIdx := l1; mIdx <= n-1; mIdx++ {
			if math.Abs(e[mIdx-1]) <= math.Sqrt(math.Abs(d[mIdx-1]))*math.Sqrt(math.Abs(d[mIdx]))*eps {
				e[mIdx-1] = 0
				mm = mIdx
				goto found
			}
		}
	found:
		l := l1
		lsv := l
		lend := mm
		lendsv := lend
		l1 = mm + 1
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
			dlasclG(anorm, ssfmax, lend-l, 1, e[l-1:], n)
		} else if anorm < ssfmin {
			iscale = 2
			dlasclG(anorm, ssfmin, lend-l+1, 1, d[l-1:], n)
			dlasclG(anorm, ssfmin, lend-l, 1, e[l-1:], n)
		}

		// Square the off-diagonals.
		for i := l; i <= lend-1; i++ {
			e[i-1] = e[i-1] * e[i-1]
		}

		// Choose QL or QR.
		if math.Abs(d[lend-1]) < math.Abs(d[l-1]) {
			lend = lsv
			l = lendsv
		}

		if lend >= l {
			// QL Iteration.
		qlLoop:
			for {
				if l != lend {
					mIdx := lend
					for k := l; k <= lend-1; k++ {
						if math.Abs(e[k-1]) <= eps2*math.Abs(d[k-1]*d[k]) {
							mIdx = k
							goto qlSplit
						}
					}
				qlSplit:
					if mIdx < lend {
						e[mIdx-1] = 0
					}
					p := d[l-1]
					if mIdx == l {
						// Eigenvalue found.
						d[l-1] = p
						l++
						if l <= lend {
							continue qlLoop
						}
						break qlLoop
					}
					if mIdx == l+1 {
						rte := math.Sqrt(e[l-1])
						rt1, rt2 := dlae2(d[l-1], rte, d[l])
						d[l-1] = rt1
						d[l] = rt2
						e[l-1] = 0
						l += 2
						if l <= lend {
							continue qlLoop
						}
						break qlLoop
					}
					if jtot == nmaxit {
						break qlLoop
					}
					jtot++

					rte := math.Sqrt(e[l-1])
					sigma := (d[l] - p) / (2 * rte)
					r := dlapy2(sigma, 1)
					sigma = p - (rte / (sigma + math.Copysign(r, sigma)))

					c := 1.0
					s := 0.0
					gamma := d[mIdx-1] - sigma
					p = gamma * gamma

					for i := mIdx - 1; i >= l; i-- {
						bb := e[i-1]
						r := p + bb
						if i != mIdx-1 {
							e[i] = s * r
						}
						oldc := c
						c = p / r
						s = bb / r
						oldgam := gamma
						alpha := d[i-1]
						gamma = c*(alpha-sigma) - s*oldgam
						d[i] = oldgam + (alpha - gamma)
						if c != 0 {
							p = (gamma * gamma) / c
						} else {
							p = oldc * bb
						}
					}
					e[l-1] = s * p
					d[l-1] = sigma + gamma
					continue qlLoop
				} else {
					// l == lend: trivial.
					p := d[l-1]
					d[l-1] = p
					l++
					if l <= lend {
						continue qlLoop
					}
					break qlLoop
				}
			}
		} else {
			// QR Iteration.
		qrLoop:
			for {
				mIdx := lend
				for k := l; k >= lend+1; k-- {
					if math.Abs(e[k-2]) <= eps2*math.Abs(d[k-1]*d[k-2]) {
						mIdx = k
						goto qrSplit
					}
				}
			qrSplit:
				if mIdx > lend {
					e[mIdx-2] = 0
				}
				p := d[l-1]
				if mIdx == l {
					// Eigenvalue found.
					d[l-1] = p
					l--
					if l >= lend {
						continue qrLoop
					}
					break qrLoop
				}
				if mIdx == l-1 {
					rte := math.Sqrt(e[l-2])
					rt1, rt2 := dlae2(d[l-1], rte, d[l-2])
					d[l-1] = rt1
					d[l-2] = rt2
					e[l-2] = 0
					l -= 2
					if l >= lend {
						continue qrLoop
					}
					break qrLoop
				}
				if jtot == nmaxit {
					break qrLoop
				}
				jtot++

				rte := math.Sqrt(e[l-2])
				sigma := (d[l-2] - p) / (2 * rte)
				r := dlapy2(sigma, 1)
				sigma = p - (rte / (sigma + math.Copysign(r, sigma)))

				c := 1.0
				s := 0.0
				gamma := d[mIdx-1] - sigma
				p = gamma * gamma

				for i := mIdx; i <= l-1; i++ {
					bb := e[i-1]
					r := p + bb
					if i != mIdx {
						e[i-2] = s * r
					}
					oldc := c
					c = p / r
					s = bb / r
					oldgam := gamma
					alpha := d[i]
					gamma = c*(alpha-sigma) - s*oldgam
					d[i-1] = oldgam + (alpha - gamma)
					if c != 0 {
						p = (gamma * gamma) / c
					} else {
						p = oldc * bb
					}
				}
				e[l-2] = s * p
				d[l-1] = sigma + gamma
				continue qrLoop
			}
		}

		// Undo scaling for this block.
		if iscale == 1 {
			dlasclG(ssfmax, anorm, lendsv-lsv+1, 1, d[lsv-1:], n)
		}
		if iscale == 2 {
			dlasclG(ssfmin, anorm, lendsv-lsv+1, 1, d[lsv-1:], n)
		}

		if jtot >= nmaxit {
			// Count failures and exit.
			for i := 1; i <= n-1; i++ {
				if e[i-1] != 0 {
					info++
				}
			}
			return info
		}
	}

	// Sort eigenvalues ascending.
	dlasrt('I', n, d)
	return 0
}
