// fa/lbfgsb_cauchy.go
//
// Port of subroutine cauchy in lbfgsb.f v3.0. Computes the generalized
// Cauchy point (the first local minimizer of the quadratic model along
// the projected steepest-descent path P(x - t·g, l, u)).
package fa

import "math"

// cauchy computes the GCP. Mirrors the Fortran 1995 implementation
// step-for-step. xcp is filled with the GCP, c with W'·(xcp − x), nseg
// is the number of intervals explored, and info is non-zero if the
// embedded bmv triangular solve goes singular.
//
// All matrix arguments are column-major flattened []float64; ws/wy have
// leading dim n with `col` columns, sy/wt have leading dim m.
func cauchy(n int, x, l, u []float64, nbd []int, g []float64,
	iorder []int, iwhere []int,
	t, d, xcp []float64,
	m int, wy, ws, sy, wt []float64,
	theta float64, col, head int,
	p, c, wbp, v []float64,
	sbgnrm, epsmch float64,
) (nseg, info int) {
	if sbgnrm <= 0 {
		dcopy(n, x, 1, xcp, 1)
		return 0, 0
	}

	bnded := true
	nfree := n + 1
	nbreak := 0
	ibkmin := 0
	bkmin := 0.0
	col2 := 2 * col
	f1 := 0.0

	// p starts as 0 and accumulates W'·d.
	for i := 0; i < col2; i++ {
		p[i] = 0
	}

	for i := 1; i <= n; i++ {
		neggi := -g[i-1]
		if iwhere[i-1] != 3 && iwhere[i-1] != -1 {
			var tl, tu float64
			if nbd[i-1] <= 2 {
				tl = x[i-1] - l[i-1]
			}
			if nbd[i-1] >= 2 {
				tu = u[i-1] - x[i-1]
			}
			xlower := nbd[i-1] <= 2 && tl <= 0
			xupper := nbd[i-1] >= 2 && tu <= 0
			iwhere[i-1] = 0
			switch {
			case xlower:
				if neggi <= 0 {
					iwhere[i-1] = 1
				}
			case xupper:
				if neggi >= 0 {
					iwhere[i-1] = 2
				}
			default:
				if math.Abs(neggi) <= 0 {
					iwhere[i-1] = -3
				}
			}
		}
		pointr := head
		if iwhere[i-1] != 0 && iwhere[i-1] != -1 {
			d[i-1] = 0
		} else {
			d[i-1] = neggi
			f1 -= neggi * neggi
			// p := p - W'·e_i · g_i  (BLNZ uses p := p + W'·e_i · neggi)
			for j := 1; j <= col; j++ {
				p[j-1] += wy[(pointr-1)*n+(i-1)] * neggi
				p[col+j-1] += ws[(pointr-1)*n+(i-1)] * neggi
				pointr = (pointr)%m + 1
			}
			switch {
			case nbd[i-1] <= 2 && nbd[i-1] != 0 && neggi < 0:
				nbreak++
				iorder[nbreak-1] = i
				t[nbreak-1] = (x[i-1] - l[i-1]) / -neggi
				if nbreak == 1 || t[nbreak-1] < bkmin {
					bkmin = t[nbreak-1]
					ibkmin = nbreak
				}
			case nbd[i-1] >= 2 && neggi > 0:
				nbreak++
				iorder[nbreak-1] = i
				t[nbreak-1] = (u[i-1] - x[i-1]) / neggi
				if nbreak == 1 || t[nbreak-1] < bkmin {
					bkmin = t[nbreak-1]
					ibkmin = nbreak
				}
			default:
				nfree--
				iorder[nfree-1] = i
				if math.Abs(neggi) > 0 {
					bnded = false
				}
			}
		}
	}

	if theta != 1 {
		dscal(col, theta, p[col:], 1)
	}

	dcopy(n, x, 1, xcp, 1)

	if nbreak == 0 && nfree == n+1 {
		// d = 0 (or only at-bound vars), GCP = x.
		return 1, 0
	}

	for j := 0; j < col2; j++ {
		c[j] = 0
	}

	f2 := -theta * f1
	f2Org := f2
	if col > 0 {
		if info = bmv(m, sy, wt, col, p, v); info != 0 {
			return 1, info
		}
		f2 -= ddot(col2, v, 1, p, 1)
	}
	dtm := -f1 / f2
	tsum := 0.0
	nseg = 1

	if nbreak == 0 {
		goto applyGCP
	}

	// ---- main loop over breakpoints ----
	{
		nleft := nbreak
		iter := 1
		tj := 0.0
		for {
			tj0 := tj
			var ibp int
			if iter == 1 {
				tj = bkmin
				ibp = iorder[ibkmin-1]
			} else {
				if iter == 2 {
					if ibkmin != nbreak {
						t[ibkmin-1] = t[nbreak-1]
						iorder[ibkmin-1] = iorder[nbreak-1]
					}
				}
				hpsolb(nleft, t, iorder, iter-2)
				tj = t[nleft-1]
				ibp = iorder[nleft-1]
			}
			dt := tj - tj0

			if dtm < dt {
				goto applyGCP
			}

			tsum += dt
			nleft--
			iter++
			dibp := d[ibp-1]
			d[ibp-1] = 0
			var zibp float64
			if dibp > 0 {
				zibp = u[ibp-1] - x[ibp-1]
				xcp[ibp-1] = u[ibp-1]
				iwhere[ibp-1] = 2
			} else {
				zibp = l[ibp-1] - x[ibp-1]
				xcp[ibp-1] = l[ibp-1]
				iwhere[ibp-1] = 1
			}
			if nleft == 0 && nbreak == n {
				dtm = dt
				goto cFinal
			}

			nseg++
			dibp2 := dibp * dibp
			// Fortran: f1 = f1 + dt*f2 + dibp2 - theta*dibp*zibp
			// left-associative — f1 first; replicate exact accumulation order.
			f1 += dt * f2
			f1 += dibp2
			f1 -= theta * dibp * zibp
			f2 -= theta * dibp2

			if col > 0 {
				daxpy(col2, dt, p, 1, c, 1)
				pointr := head
				for j := 1; j <= col; j++ {
					wbp[j-1] = wy[(pointr-1)*n+(ibp-1)]
					wbp[col+j-1] = theta * ws[(pointr-1)*n+(ibp-1)]
					pointr = (pointr)%m + 1
				}
				if info = bmv(m, sy, wt, col, wbp, v); info != 0 {
					return nseg, info
				}
				wmc := ddot(col2, c, 1, v, 1)
				wmp := ddot(col2, p, 1, v, 1)
				wmw := ddot(col2, wbp, 1, v, 1)
				daxpy(col2, -dibp, wbp, 1, p, 1)
				f1 += dibp * wmc
				// Fortran: f2 = f2 + 2.0d0*dibp*wmp - dibp2*wmw — left-fold.
				f2 += 2.0 * dibp * wmp
				f2 -= dibp2 * wmw
			}
			if v := epsmch * f2Org; f2 < v {
				f2 = v
			}
			if nleft > 0 {
				dtm = -f1 / f2
				continue
			} else if bnded {
				f1 = 0
				f2 = 0
				dtm = 0
			} else {
				dtm = -f1 / f2
			}
			break
		}
	}

applyGCP:
	if dtm <= 0 {
		dtm = 0
	}
	tsum += dtm
	// Move free variables and the ones whose breakpoints we never reached.
	daxpy(n, tsum, d, 1, xcp, 1)

cFinal:
	if col > 0 {
		daxpy(col2, dtm, p, 1, c, 1)
	}
	return nseg, 0
}
