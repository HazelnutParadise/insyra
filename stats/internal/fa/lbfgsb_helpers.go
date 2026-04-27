// fa/lbfgsb_helpers.go
//
// Direct ports of the small bookkeeping subroutines from lbfgsb.f v3.0:
//   active   — initial active-set classification + projection
//   projgr   — infinity norm of the projected gradient
//   errclb   — input error checking
//   freev    — find free / leaving / entering variables at the GCP
//   hpsolb   — heap-based pop of the smallest breakpoint
//   matupd   — update the (s, y) pairs and the SY/SS matrices
//   cmprlb   — compute the reduced gradient r = -Z'B(xcp - xk) - Z'g
//   lnsrlb   — driver for the Moré-Thuente line search
//   bmv      — multiply the 2m × 2m middle matrix in the compact L-BFGS form
//
// All routines preserve Fortran's 1-based semantics through explicit
// `idx-1` adjustments at access sites; matrix arguments are column-major
// flattened []float64 with leading dimension `lda`.
package fa

import (
	"math"
)

// active initialises iwhere and projects x into the box.
//
// nbd[i]: 0 unbounded, 1 lower only, 2 both, 3 upper only.
// On exit:
//
//	iwhere[i] = -1  if x_i has no bounds
//	             3  if l_i = u_i (always fixed)
//	             0  otherwise (free with bounds)
//
// prjctd, cnstnd, boxed are diagnostic flags consumed downstream.
func active(n int, l, u []float64, nbd []int, x []float64, iwhere []int) (prjctd, cnstnd, boxed bool) {
	prjctd = false
	cnstnd = false
	boxed = true

	for i := 0; i < n; i++ {
		if nbd[i] > 0 {
			if nbd[i] <= 2 && x[i] <= l[i] {
				if x[i] < l[i] {
					prjctd = true
					x[i] = l[i]
				}
			} else if nbd[i] >= 2 && x[i] >= u[i] {
				if x[i] > u[i] {
					prjctd = true
					x[i] = u[i]
				}
			}
		}
	}
	for i := 0; i < n; i++ {
		if nbd[i] != 2 {
			boxed = false
		}
		if nbd[i] == 0 {
			iwhere[i] = -1
		} else {
			cnstnd = true
			if nbd[i] == 2 && u[i]-l[i] <= 0 {
				iwhere[i] = 3
			} else {
				iwhere[i] = 0
			}
		}
	}
	return
}

// projgr returns the infinity norm of the projected gradient.
func projgr(n int, l, u []float64, nbd []int, x, g []float64) float64 {
	sbgnrm := 0.0
	for i := 0; i < n; i++ {
		gi := g[i]
		if nbd[i] != 0 {
			if gi < 0 {
				if nbd[i] >= 2 {
					if v := x[i] - u[i]; v > gi {
						gi = v
					}
				}
			} else {
				if nbd[i] <= 2 {
					if v := x[i] - l[i]; v < gi {
						gi = v
					}
				}
			}
		}
		if a := math.Abs(gi); a > sbgnrm {
			sbgnrm = a
		}
	}
	return sbgnrm
}

// errclb validates the inputs. Returns task ("" on success), info, k.
func errclb(n, m int, factr float64, l, u []float64, nbd []int) (task string, info, k int) {
	if n <= 0 {
		return "ERROR: N .LE. 0", 0, 0
	}
	if m <= 0 {
		return "ERROR: M .LE. 0", 0, 0
	}
	if factr < 0 {
		return "ERROR: FACTR .LT. 0", 0, 0
	}
	for i := 0; i < n; i++ {
		if nbd[i] < 0 || nbd[i] > 3 {
			return "ERROR: INVALID NBD", -6, i + 1
		}
		if nbd[i] == 2 && l[i] > u[i] {
			return "ERROR: NO FEASIBLE SOLUTION", -7, i + 1
		}
	}
	return "", 0, 0
}

// freev counts entering / leaving variables and partitions index into
// (free | active) blocks. Mirrors the Fortran routine including its 1-based
// indexing semantics: index slots 0..nfree-1 hold free vars (1-based ids),
// slots nfree..n-1 hold the active ones.
func freev(n int, nfree *int, index []int, nenter, ileave *int, indx2 []int,
	iwhere []int, updatd, cnstnd bool, iter int,
) (wrk bool) {
	*nenter = 0
	*ileave = n + 1
	if iter > 0 && cnstnd {
		for i := 1; i <= *nfree; i++ {
			k := index[i-1]
			if iwhere[k-1] > 0 {
				*ileave--
				indx2[*ileave-1] = k
			}
		}
		for i := *nfree + 1; i <= n; i++ {
			k := index[i-1]
			if iwhere[k-1] <= 0 {
				*nenter++
				indx2[*nenter-1] = k
			}
		}
	}
	wrk = (*ileave < n+1) || (*nenter > 0) || updatd

	*nfree = 0
	iact := n + 1
	for i := 1; i <= n; i++ {
		if iwhere[i-1] <= 0 {
			*nfree++
			index[*nfree-1] = i
		} else {
			iact--
			index[iact-1] = i
		}
	}
	return wrk
}

// hpsolb pops the least element of t to position n-1 and rearranges the
// remainder as a min-heap. iheap=0 forces an initial heapify.
//
// Operates on 1..n (Fortran semantics) but with 0-based slices: t[0..n-1].
func hpsolb(n int, t []float64, iorder []int, iheap int) {
	if iheap == 0 {
		for k := 2; k <= n; k++ {
			ddum := t[k-1]
			indxin := iorder[k-1]
			i := k
			for i > 1 {
				j := i / 2
				if ddum < t[j-1] {
					t[i-1] = t[j-1]
					iorder[i-1] = iorder[j-1]
					i = j
					continue
				}
				break
			}
			t[i-1] = ddum
			iorder[i-1] = indxin
		}
	}
	if n > 1 {
		i := 1
		out := t[0]
		indxou := iorder[0]
		ddum := t[n-1]
		indxin := iorder[n-1]
		for {
			j := i + i
			if j <= n-1 {
				if t[j+1-1] < t[j-1] {
					j++
				}
				if t[j-1] < ddum {
					t[i-1] = t[j-1]
					iorder[i-1] = iorder[j-1]
					i = j
					continue
				}
			}
			break
		}
		t[i-1] = ddum
		iorder[i-1] = indxin
		t[n-1] = out
		iorder[n-1] = indxou
	}
}

// matupd updates the WS, WY, SY, SS matrices after a successful step.
//
// ws / wy are n x m column-major; sy / ss are m x m column-major. itail/col/
// head are mutated to reflect the rolling buffer state.
func matupd(n, m int, ws, wy, sy, ss []float64, d, r []float64,
	itail, iupdat, col, head *int,
	theta *float64, rr, dr, stp, dtd float64,
) {
	if *iupdat <= m {
		*col = *iupdat
		*itail = ((*head+*iupdat-2)%m + m) % m + 1
	} else {
		*itail = (*itail)%m + 1
		*head = (*head)%m + 1
	}
	// Copy d into ws(:, itail) and r into wy(:, itail) (column itail, 1-based).
	dcopy(n, d, 1, ws[(*itail-1)*n:], 1)
	dcopy(n, r, 1, wy[(*itail-1)*n:], 1)

	*theta = rr / dr

	// If we wrapped, shift columns 2..col of SS up-left and rows 2..col of
	// SY similarly (ss(2,j+1) -> ss(1,j); sy(j+1,j+1) -> sy(j,j)).
	if *iupdat > m {
		for j := 1; j <= *col-1; j++ {
			// dcopy(j, ss(2,j+1), 1, ss(1,j), 1):
			//   from column j+1, rows 2..2+j-1  →  column j, rows 1..j
			src := (j+1-1)*m + (2 - 1)
			dst := (j-1)*m + (1 - 1)
			dcopy(j, ss[src:], 1, ss[dst:], 1)
			// dcopy(col-j, sy(j+1,j+1), 1, sy(j,j), 1):
			//   from column j+1, rows j+1..col  →  column j, rows j..col-1
			src = (j+1-1)*m + (j + 1 - 1)
			dst = (j-1)*m + (j - 1)
			dcopy(*col-j, sy[src:], 1, sy[dst:], 1)
		}
	}

	// Add new row of SY and new column of SS.
	pointr := *head
	for j := 1; j <= *col-1; j++ {
		// sy(col, j) = ddot(n, d, 1, wy(:, pointr), 1)
		sy[(j-1)*m+(*col-1)] = ddot(n, d, 1, wy[(pointr-1)*n:], 1)
		// ss(j, col) = ddot(n, ws(:, pointr), 1, d, 1)
		ss[(*col-1)*m+(j-1)] = ddot(n, ws[(pointr-1)*n:], 1, d, 1)
		pointr = (pointr)%m + 1
	}
	if stp == 1 {
		ss[(*col-1)*m+(*col-1)] = dtd
	} else {
		ss[(*col-1)*m+(*col-1)] = stp * stp * dtd
	}
	sy[(*col-1)*m+(*col-1)] = dr
}

// cmprlb computes r = -Z'B(xcp - xk) - Z'g using wa(2m+1) = W'(xcp - x)
// produced by cauchy. Returns info: 0 on success, -8 if bmv failed.
func cmprlb(n, m int, x, g, ws, wy, sy, wt, z, r, wa []float64,
	index []int, theta float64, col, head, nfree int, cnstnd bool,
) (info int) {
	if !cnstnd && col > 0 {
		for i := 0; i < n; i++ {
			r[i] = -g[i]
		}
		return 0
	}
	for i := 1; i <= nfree; i++ {
		k := index[i-1]
		r[i-1] = -theta*(z[k-1]-x[k-1]) - g[k-1]
	}
	if info = bmv(m, sy, wt, col, wa[2*m:], wa[:]); info != 0 {
		return -8
	}
	pointr := head
	for j := 1; j <= col; j++ {
		a1 := wa[j-1]
		a2 := theta * wa[col+j-1]
		for i := 1; i <= nfree; i++ {
			k := index[i-1]
			r[i-1] += wy[(pointr-1)*n+(k-1)]*a1 + ws[(pointr-1)*n+(k-1)]*a2
		}
		pointr = (pointr)%m + 1
	}
	return 0
}

// bmv multiplies the 2m × 2m middle matrix M of the compact L-BFGS form
// by a 2m vector v, putting the product in p. info != 0 if the embedded
// triangular solve fails. Direct port of the Fortran subroutine.
func bmv(m int, sy, wt []float64, col int, v, p []float64) (info int) {
	if col == 0 {
		return 0
	}
	// Solve J * p2 = v2 + L*D^{-1}*v1.
	p[col+1-1] = v[col+1-1]
	for i := 2; i <= col; i++ {
		i2 := col + i
		sum := 0.0
		for k := 1; k <= i-1; k++ {
			sum += sy[(k-1)*m+(i-1)] * v[k-1] / sy[(k-1)*m+(k-1)]
		}
		p[i2-1] = v[i2-1] + sum
	}
	if info = dtrsl(wt, m, col, p[col:], 11); info != 0 {
		return info
	}
	// p1 = v / sqrt(diag(D))
	for i := 1; i <= col; i++ {
		p[i-1] = v[i-1] / math.Sqrt(sy[(i-1)*m+(i-1)])
	}
	// Part II: solve J' * p2 = p2.
	if info = dtrsl(wt, m, col, p[col:], 1); info != 0 {
		return info
	}
	// p1 = -D^{-1/2} * p1
	for i := 1; i <= col; i++ {
		p[i-1] = -p[i-1] / math.Sqrt(sy[(i-1)*m+(i-1)])
	}
	// p1 = p1 + D^{-1} * L' * p2
	for i := 1; i <= col; i++ {
		sum := 0.0
		for k := i + 1; k <= col; k++ {
			sum += sy[(i-1)*m+(k-1)] * p[col+k-1] / sy[(i-1)*m+(i-1)]
		}
		p[i-1] += sum
	}
	return 0
}

// lnsrlbState mirrors the Fortran isave[2] + dsave[13] persistence used by
// dcsrch through lnsrlb. Plus the bookkeeping mainlb keeps across reentry.
type lnsrlbState struct {
	dcs   dcsrchState
	csave string
}

// lnsrlb performs the Moré-Thuente line search with safeguarding so all
// trial points lie within the feasible region. Returns a continuation
// task: "FG_LNSRCH" if the caller needs to evaluate f, g at x and re-enter,
// or "NEW_X" once the line search succeeds. info != 0 is a fatal abort.
func lnsrlb(n int, l, u []float64, nbd []int,
	x []float64, f, fold, gd, gdold *float64, g, d, r, t, z []float64,
	stp, dnorm, dtd, xstep, stpmx *float64,
	iter int, ifun, iback, nfgv *int, info *int,
	task string, boxed, cnstnd bool,
	st *lnsrlbState,
) string {
	const (
		one  = 1.0
		zero = 0.0
		big  = 1e10
		ftol = 1e-3
		gtol = 0.9
		xtol = 0.1
	)

	if !(len(task) >= 5 && task[:5] == "FG_LN") {
		// First entry for this iteration: prepare the search direction.
		*dtd = ddot(n, d, 1, d, 1)
		*dnorm = math.Sqrt(*dtd)

		*stpmx = big
		if cnstnd {
			if iter == 0 {
				*stpmx = one
			} else {
				for i := 0; i < n; i++ {
					a1 := d[i]
					if nbd[i] != 0 {
						if a1 < 0 && nbd[i] <= 2 {
							a2 := l[i] - x[i]
							switch {
							case a2 >= 0:
								*stpmx = 0
							case a1*(*stpmx) < a2:
								*stpmx = a2 / a1
							}
						} else if a1 > 0 && nbd[i] >= 2 {
							a2 := u[i] - x[i]
							switch {
							case a2 <= 0:
								*stpmx = 0
							case a1*(*stpmx) > a2:
								*stpmx = a2 / a1
							}
						}
					}
				}
			}
		}

		if iter == 0 && !boxed {
			*stp = math.Min(one/(*dnorm), *stpmx)
		} else {
			*stp = one
		}

		dcopy(n, x, 1, t, 1)
		dcopy(n, g, 1, r, 1)
		*fold = *f
		*ifun = 0
		*iback = 0
		st.csave = "START"
	}

	*gd = ddot(n, g, 1, d, 1)
	if *ifun == 0 {
		*gdold = *gd
		if *gd >= zero {
			// directional derivative >= 0; line search impossible
			*info = -4
			return task
		}
	}

	dcsrch(f, gd, stp, ftol, gtol, xtol, zero, *stpmx, &st.csave, &st.dcs)

	*xstep = (*stp) * (*dnorm)
	if !(len(st.csave) >= 4 && (st.csave[:4] == "CONV" || st.csave[:4] == "WARN")) {
		*ifun++
		*nfgv++
		*iback = *ifun - 1
		if *stp == one {
			dcopy(n, z, 1, x, 1)
		} else {
			for i := 0; i < n; i++ {
				x[i] = (*stp)*d[i] + t[i]
			}
		}
		return "FG_LNSRCH"
	}
	return "NEW_X"
}
