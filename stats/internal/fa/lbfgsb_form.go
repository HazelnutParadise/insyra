// fa/lbfgsb_form.go
//
// Ports of subroutines formk and formt from lbfgsb.f v3.0. These build
// and Cholesky-factorize the middle matrices of the compact L-BFGS form
// used by the subspace minimization (formk) and the Cauchy point (formt).
package fa

// formk builds the LEL^T factorization of
//
//	K = [-D − Y'ZZ'Y/theta     L_a' − R_z']
//	    [L_a − R_z              theta·S'AA'S]
//
// stored in `wn` (2m × 2m). `wn1` retains the unsigned version
// [Y'ZZ'Y, L_a' + R_z'; L_a + R_z, S'AA'S] for incremental update on the
// next call. info encodes Cholesky failure: -1 in (1,1) block, -2 in (2,2).
func formk(n, nsub int, ind []int, nenter, ileave int, indx2 []int,
	iupdat int, updatd bool,
	wn, wn1 []float64, m int,
	ws, wy, sy []float64,
	theta float64, col, head int,
) (info int) {
	// "ld" leading dim of wn / wn1.
	const _ = 0 // (no extra constants)
	ldwn := 2 * m

	// 1. Update wn1 with new (1,1), (2,2), (2,1) blocks if updatd.
	if updatd {
		if iupdat > m {
			// Shift old part of wn1.
			for jy := 1; jy <= m-1; jy++ {
				js := m + jy
				// dcopy(m-jy, wn1(jy+1,jy+1), 1, wn1(jy,jy), 1)
				dcopy(m-jy,
					wn1[(jy+1-1)*ldwn+(jy+1-1):], 1,
					wn1[(jy-1)*ldwn+(jy-1):], 1,
				)
				dcopy(m-jy,
					wn1[(js+1-1)*ldwn+(js+1-1):], 1,
					wn1[(js-1)*ldwn+(js-1):], 1,
				)
				dcopy(m-1,
					wn1[(jy+1-1)*ldwn+(m+2-1):], 1,
					wn1[(jy-1)*ldwn+(m+1-1):], 1,
				)
			}
		}
		pbegin := 1
		pend := nsub
		dbegin := nsub + 1
		dend := n
		iy := col
		is := m + col
		ipntr := head + col - 1
		if ipntr > m {
			ipntr -= m
		}
		jpntr := head
		for jy := 1; jy <= col; jy++ {
			js := m + jy
			temp1, temp2, temp3 := 0.0, 0.0, 0.0
			for k := pbegin; k <= pend; k++ {
				k1 := ind[k-1]
				temp1 += wy[(ipntr-1)*n+(k1-1)] * wy[(jpntr-1)*n+(k1-1)]
			}
			for k := dbegin; k <= dend; k++ {
				k1 := ind[k-1]
				temp2 += ws[(ipntr-1)*n+(k1-1)] * ws[(jpntr-1)*n+(k1-1)]
				temp3 += ws[(ipntr-1)*n+(k1-1)] * wy[(jpntr-1)*n+(k1-1)]
			}
			wn1[(jy-1)*ldwn+(iy-1)] = temp1
			wn1[(js-1)*ldwn+(is-1)] = temp2
			wn1[(jy-1)*ldwn+(is-1)] = temp3
			jpntr = (jpntr)%m + 1
		}

		// New column in block (2,1).
		jy := col
		jpntr = head + col - 1
		if jpntr > m {
			jpntr -= m
		}
		ipntr = head
		for i := 1; i <= col; i++ {
			is := m + i
			temp3 := 0.0
			for k := pbegin; k <= pend; k++ {
				k1 := ind[k-1]
				temp3 += ws[(ipntr-1)*n+(k1-1)] * wy[(jpntr-1)*n+(k1-1)]
			}
			ipntr = (ipntr)%m + 1
			wn1[(jy-1)*ldwn+(is-1)] = temp3
		}
	}
	upcl := col
	if updatd {
		upcl = col - 1
	}

	// 2. Modify old (1,1) and (2,2) blocks for entering / leaving variables.
	{
		ipntr := head
		for iy := 1; iy <= upcl; iy++ {
			is := m + iy
			jpntr := head
			for jy := 1; jy <= iy; jy++ {
				js := m + jy
				temp1, temp2, temp3, temp4 := 0.0, 0.0, 0.0, 0.0
				for k := 1; k <= nenter; k++ {
					k1 := indx2[k-1]
					temp1 += wy[(ipntr-1)*n+(k1-1)] * wy[(jpntr-1)*n+(k1-1)]
					temp2 += ws[(ipntr-1)*n+(k1-1)] * ws[(jpntr-1)*n+(k1-1)]
				}
				for k := ileave; k <= n; k++ {
					k1 := indx2[k-1]
					temp3 += wy[(ipntr-1)*n+(k1-1)] * wy[(jpntr-1)*n+(k1-1)]
					temp4 += ws[(ipntr-1)*n+(k1-1)] * ws[(jpntr-1)*n+(k1-1)]
				}
				wn1[(jy-1)*ldwn+(iy-1)] += temp1 - temp3
				wn1[(js-1)*ldwn+(is-1)] += -temp2 + temp4
				jpntr = (jpntr)%m + 1
			}
			ipntr = (ipntr)%m + 1
		}

		// 3. Modify old (2,1) block.
		ipntr = head
		for is := m + 1; is <= m+upcl; is++ {
			jpntr := head
			for jy := 1; jy <= upcl; jy++ {
				temp1, temp3 := 0.0, 0.0
				for k := 1; k <= nenter; k++ {
					k1 := indx2[k-1]
					temp1 += ws[(ipntr-1)*n+(k1-1)] * wy[(jpntr-1)*n+(k1-1)]
				}
				for k := ileave; k <= n; k++ {
					k1 := indx2[k-1]
					temp3 += ws[(ipntr-1)*n+(k1-1)] * wy[(jpntr-1)*n+(k1-1)]
				}
				if is <= jy+m {
					wn1[(jy-1)*ldwn+(is-1)] += temp1 - temp3
				} else {
					wn1[(jy-1)*ldwn+(is-1)] += -temp1 + temp3
				}
				jpntr = (jpntr)%m + 1
			}
			ipntr = (ipntr)%m + 1
		}
	}

	// 4. Form upper triangle of WN.
	m2 := 2 * m
	_ = m2
	for iy := 1; iy <= col; iy++ {
		is := col + iy
		is1 := m + iy
		for jy := 1; jy <= iy; jy++ {
			js := col + jy
			js1 := m + jy
			wn[(iy-1)*ldwn+(jy-1)] = wn1[(jy-1)*ldwn+(iy-1)] / theta
			wn[(is-1)*ldwn+(js-1)] = wn1[(js1-1)*ldwn+(is1-1)] * theta
		}
		for jy := 1; jy <= iy-1; jy++ {
			wn[(is-1)*ldwn+(jy-1)] = -wn1[(jy-1)*ldwn+(is1-1)]
		}
		for jy := iy; jy <= col; jy++ {
			wn[(is-1)*ldwn+(jy-1)] = wn1[(jy-1)*ldwn+(is1-1)]
		}
		wn[(iy-1)*ldwn+(iy-1)] += sy[(iy-1)*m+(iy-1)]
	}

	// 5. Cholesky-factorize the (1,1) block. dpofa works on the leading
	// `col`×`col` submatrix with leading dim ldwn = 2m.
	if k := dpofa(wn, ldwn, col); k != 0 {
		return -1
	}
	col2 := 2 * col
	for js := col + 1; js <= col2; js++ {
		// dtrsl(wn, m2, col, wn(1, js), 11, info)
		dtrsl(wn, ldwn, col, wn[(js-1)*ldwn:], 11)
	}

	// 6. Update (2,2) block: WN[is,js] += <WN[1..col, is], WN[1..col, js]>.
	for is := col + 1; is <= col2; is++ {
		for js := is; js <= col2; js++ {
			wn[(js-1)*ldwn+(is-1)] += ddot(col, wn[(is-1)*ldwn:], 1, wn[(js-1)*ldwn:], 1)
		}
	}

	// 7. Cholesky factor (2,2) block. The Fortran call is
	//    dpofa(wn(col+1, col+1), m2, col, info)
	// which is the submatrix anchored at (col+1, col+1) with leading dim m2.
	sub := wn[(col)*ldwn+(col):]
	if k := dpofa(sub, ldwn, col); k != 0 {
		return -2
	}
	return 0
}

// formt builds T = theta·SS + L·D^{-1}·L', stores in upper triangle of wt
// and Cholesky-factorizes it. info=-3 on failure.
func formt(m int, wt, sy, ss []float64, col int, theta float64) (info int) {
	for j := 1; j <= col; j++ {
		wt[(j-1)*m+(1-1)] = theta * ss[(j-1)*m+(1-1)]
	}
	for i := 2; i <= col; i++ {
		for j := i; j <= col; j++ {
			k1 := i
			if j < i {
				k1 = j
			}
			k1--
			ddum := 0.0
			for k := 1; k <= k1; k++ {
				ddum += sy[(k-1)*m+(i-1)] * sy[(k-1)*m+(j-1)] / sy[(k-1)*m+(k-1)]
			}
			wt[(j-1)*m+(i-1)] = ddum + theta*ss[(j-1)*m+(i-1)]
		}
	}
	if k := dpofa(wt, m, col); k != 0 {
		return -3
	}
	return 0
}
