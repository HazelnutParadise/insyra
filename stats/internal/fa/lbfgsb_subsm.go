// fa/lbfgsb_subsm.go
//
// Port of subroutine subsm from lbfgsb.f v3.0 (with the Morales–Nocedal
// 2010 backtracking-projection refinement).
//
// Computes an approximate solution to the subspace problem
//
//	min  Q(x) = r'(x − xcp) + 1/2 (x − xcp)' B (x − xcp)
//	s.t. l ≤ x ≤ u, x_i = xcp_i for i ∈ A(xcp)
//
// using the compact L-BFGS direction d = (1/θ)·r + (1/θ²)·Z'·W·K^{-1}·W'·Z·r,
// then safeguarding by projection / backtracking.
package fa

// subsm computes the (subspace) Newton point and writes it to x. d holds
// the reduced gradient on entry and the Newton direction on exit.
//
// iword = 0 if the projected step stays in the box, 1 otherwise.
// info != 0 on a failed triangular solve.
func subsm(n, m, nsub int, ind []int, l, u []float64, nbd []int,
	x, d, xp, ws, wy []float64,
	theta float64, xx, gg []float64,
	col, head int,
	wv, wn []float64,
) (iword, info int) {
	if nsub <= 0 {
		return 0, 0
	}
	ldwn := 2 * m

	// Compute wv = W'·Z·d.
	pointr := head
	for i := 1; i <= col; i++ {
		temp1, temp2 := 0.0, 0.0
		for j := 1; j <= nsub; j++ {
			k := ind[j-1]
			temp1 += wy[(pointr-1)*n+(k-1)] * d[j-1]
			temp2 += ws[(pointr-1)*n+(k-1)] * d[j-1]
		}
		wv[i-1] = temp1
		wv[col+i-1] = theta * temp2
		pointr = (pointr)%m + 1
	}

	col2 := 2 * col
	if info = dtrsl(wn, ldwn, col2, wv, 11); info != 0 {
		return 0, info
	}
	for i := 1; i <= col; i++ {
		wv[i-1] = -wv[i-1]
	}
	if info = dtrsl(wn, ldwn, col2, wv, 1); info != 0 {
		return 0, info
	}

	// d = (1/θ)·d + (1/θ²)·Z'·W·wv.
	pointr = head
	for jy := 1; jy <= col; jy++ {
		js := col + jy
		for i := 1; i <= nsub; i++ {
			k := ind[i-1]
			// Fortran: d(i) = d(i) + wy(k,p)*wv(jy)/theta + ws(k,p)*wv(js)
			// Left-fold; replicate exact accumulation order.
			d[i-1] += wy[(pointr-1)*n+(k-1)] * wv[jy-1] / theta
			d[i-1] += ws[(pointr-1)*n+(k-1)] * wv[js-1]
		}
		pointr = (pointr)%m + 1
	}
	dscal(nsub, 1.0/theta, d, 1)

	// Projection of the Newton direction.
	iword = 0
	dcopy(n, x, 1, xp, 1)
	for i := 1; i <= nsub; i++ {
		k := ind[i-1]
		dk := d[i-1]
		xk := x[k-1]
		if nbd[k-1] != 0 {
			switch nbd[k-1] {
			case 1: // lower only
				if v := xk + dk; v > l[k-1] {
					x[k-1] = v
				} else {
					x[k-1] = l[k-1]
				}
				if x[k-1] == l[k-1] {
					iword = 1
				}
			case 2: // both
				v := xk + dk
				if v < l[k-1] {
					v = l[k-1]
				}
				if v > u[k-1] {
					v = u[k-1]
				}
				x[k-1] = v
				if x[k-1] == l[k-1] || x[k-1] == u[k-1] {
					iword = 1
				}
			case 3: // upper only
				if v := xk + dk; v < u[k-1] {
					x[k-1] = v
				} else {
					x[k-1] = u[k-1]
				}
				if x[k-1] == u[k-1] {
					iword = 1
				}
			}
		} else {
			x[k-1] = xk + dk
		}
	}
	if iword == 0 {
		return 0, 0
	}

	// Check directional derivative; if positive, fall back to backtracking.
	ddP := 0.0
	for i := 0; i < n; i++ {
		ddP += (x[i] - xx[i]) * gg[i]
	}
	if ddP <= 0 {
		return iword, 0
	}
	dcopy(n, xp, 1, x, 1)

	// Backtracking with active-set safeguarding.
	alpha := 1.0
	temp1 := alpha
	ibd := 0
	for i := 1; i <= nsub; i++ {
		k := ind[i-1]
		dk := d[i-1]
		if nbd[k-1] != 0 {
			switch {
			case dk < 0 && nbd[k-1] <= 2:
				temp2 := l[k-1] - x[k-1]
				switch {
				case temp2 >= 0:
					temp1 = 0
				case dk*alpha < temp2:
					temp1 = temp2 / dk
				}
			case dk > 0 && nbd[k-1] >= 2:
				temp2 := u[k-1] - x[k-1]
				switch {
				case temp2 <= 0:
					temp1 = 0
				case dk*alpha > temp2:
					temp1 = temp2 / dk
				}
			}
			if temp1 < alpha {
				alpha = temp1
				ibd = i
			}
		}
	}
	if alpha < 1 {
		dk := d[ibd-1]
		k := ind[ibd-1]
		if dk > 0 {
			x[k-1] = u[k-1]
			d[ibd-1] = 0
		} else if dk < 0 {
			x[k-1] = l[k-1]
			d[ibd-1] = 0
		}
	}
	for i := 1; i <= nsub; i++ {
		k := ind[i-1]
		x[k-1] += alpha * d[i-1]
	}
	return iword, 0
}
