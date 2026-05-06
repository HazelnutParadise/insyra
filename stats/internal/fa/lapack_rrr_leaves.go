// fa/lapack_rrr_leaves.go
//
// Small leaf routines underpinning the MRRR eigenvalue path:
//
//   dlarra — split a tridiagonal matrix into independent sub-blocks.
//   dlarrc — Sturm-sequence eigenvalue counting in an interval.
//   dlaruv — multiplicative congruential RNG returning N<=128 reals
//             in (0,1). Auxiliary for dlarnv.
//   dlarnv — vectorised RNG (uniform/normal) wrapping dlaruv.
//
// Faithful translations of LAPACK 3.12.1 reference Fortran.
package fa

import "math"

// dlarra computes splitting points of a symmetric tridiagonal matrix.
// Off-diagonals e[i] (and e2[i] = e[i]^2) are zeroed where they fall
// below the threshold; isplit[0..nsplit-1] then lists the split indices
// (1-based) and isplit[nsplit-1] = n.
//
// Inputs:
//
//	d, e, e2 : tridiag (e/e2 modified in-place)
//	spltol   : if <0 absolute |e| <= |spltol|*tnrm; else relative
//	           |e| <= spltol * sqrt(|d[i]| * |d[i+1]|)
//	tnrm     : ||T||_? (only used when spltol < 0)
//
// Outputs: nsplit (number of blocks), isplit (length n).
func dlarra(n int, d, e, e2 []float64, spltol, tnrm float64, isplit []int) (nsplit, info int) {
	if n <= 0 {
		return 1, 0
	}
	nsplit = 1
	if spltol < 0 {
		thr := math.Abs(spltol) * tnrm
		for i := 0; i < n-1; i++ {
			if math.Abs(e[i]) <= thr {
				e[i] = 0
				e2[i] = 0
				isplit[nsplit-1] = i + 1
				nsplit++
			}
		}
	} else {
		for i := 0; i < n-1; i++ {
			if math.Abs(e[i]) <= spltol*math.Sqrt(math.Abs(d[i]))*math.Sqrt(math.Abs(d[i+1])) {
				e[i] = 0
				e2[i] = 0
				isplit[nsplit-1] = i + 1
				nsplit++
			}
		}
	}
	isplit[nsplit-1] = n
	return nsplit, 0
}

// dlarrc returns counts (lcnt, rcnt) of Sturm-sequence sign changes for
// shifts vl and vu, plus the eigenvalue count in (vl, vu] = rcnt-lcnt.
//
// jobt:
//
//	'T' or 't' — operate on tridiagonal T with diagonal d and off-diagonal e
//	'L' or 'l' — operate on LDL^T with d=D and e=L's subdiagonal
func dlarrc(jobt byte, n int, vl, vu float64, d, e []float64, pivmin float64) (
	eigcnt, lcnt, rcnt, info int) {
	if n <= 0 {
		return 0, 0, 0, 0
	}
	matT := jobt == 'T' || jobt == 't'
	if matT {
		// Sturm sequence on T.
		lpivot := d[0] - vl
		rpivot := d[0] - vu
		if lpivot <= 0 {
			lcnt++
		}
		if rpivot <= 0 {
			rcnt++
		}
		for i := 0; i < n-1; i++ {
			tmp := e[i] * e[i]
			lpivot = (d[i+1] - vl) - tmp/lpivot
			rpivot = (d[i+1] - vu) - tmp/rpivot
			if lpivot <= 0 {
				lcnt++
			}
			if rpivot <= 0 {
				rcnt++
			}
		}
	} else {
		// Sturm sequence on L D L^T.
		sl := -vl
		su := -vu
		for i := 0; i < n-1; i++ {
			lpivot := d[i] + sl
			rpivot := d[i] + su
			if lpivot <= 0 {
				lcnt++
			}
			if rpivot <= 0 {
				rcnt++
			}
			tmp := e[i] * d[i] * e[i]
			tmp2 := tmp / lpivot
			if tmp2 == 0 {
				sl = tmp - vl
			} else {
				sl = sl*tmp2 - vl
			}
			tmp2 = tmp / rpivot
			if tmp2 == 0 {
				su = tmp - vu
			} else {
				su = su*tmp2 - vu
			}
		}
		lpivot := d[n-1] + sl
		rpivot := d[n-1] + su
		if lpivot <= 0 {
			lcnt++
		}
		if rpivot <= 0 {
			rcnt++
		}
	}
	eigcnt = rcnt - lcnt
	_ = pivmin // not used by reference; signature parity only
	return eigcnt, lcnt, rcnt, 0
}

// dlaruv multiplicative congruential RNG.
// Generates min(n, 128) reals in (0,1), updates iseed in place.
// iseed must satisfy 0 <= iseed[i] < 4096 and iseed[3] odd.
func dlaruv(iseed []int, n int, x []float64) {
	const lv = 128
	const ipw2 = 4096
	const r = 1.0 / float64(ipw2)

	if n < 1 {
		return
	}
	i1 := iseed[0]
	i2 := iseed[1]
	i3 := iseed[2]
	i4 := iseed[3]

	limit := n
	if limit > lv {
		limit = lv
	}

	var it1, it2, it3, it4 int
	for i := 0; i < limit; i++ {
		// Loop until we don't get exactly 1.0.
		for {
			it4 = i4 * dlaruvMM[i][3]
			it3 = it4 / ipw2
			it4 = it4 - ipw2*it3
			it3 = it3 + i3*dlaruvMM[i][3] + i4*dlaruvMM[i][2]
			it2 = it3 / ipw2
			it3 = it3 - ipw2*it2
			it2 = it2 + i2*dlaruvMM[i][3] + i3*dlaruvMM[i][2] + i4*dlaruvMM[i][1]
			it1 = it2 / ipw2
			it2 = it2 - ipw2*it1
			it1 = it1 + i1*dlaruvMM[i][3] + i2*dlaruvMM[i][2] + i3*dlaruvMM[i][1] + i4*dlaruvMM[i][0]
			it1 = it1 % ipw2
			x[i] = r * (float64(it1) + r*(float64(it2)+r*(float64(it3)+r*float64(it4))))
			if x[i] != 1.0 {
				break
			}
			i1 += 2
			i2 += 2
			i3 += 2
			i4 += 2
		}
	}
	iseed[0] = it1
	iseed[1] = it2
	iseed[2] = it3
	iseed[3] = it4
}

// dlarnv returns a vector of n random reals from one of three
// distributions:
//
//	idist == 1 : uniform (0, 1)
//	idist == 2 : uniform (-1, 1)
//	idist == 3 : normal  (0, 1)  via Box-Muller
func dlarnv(idist, n int, iseed []int, x []float64) {
	const lv = 128
	const halfLV = lv / 2
	const twoPi = 6.28318530717958647692528676655900576839
	u := make([]float64, lv)
	iv := 0
	for iv < n {
		il := halfLV
		if il > n-iv {
			il = n - iv
		}
		il2 := il
		if idist == 3 {
			il2 = 2 * il
		}
		dlaruv(iseed, il2, u)
		switch idist {
		case 1:
			for i := 0; i < il; i++ {
				x[iv+i] = u[i]
			}
		case 2:
			for i := 0; i < il; i++ {
				x[iv+i] = 2*u[i] - 1
			}
		case 3:
			for i := 0; i < il; i++ {
				x[iv+i] = math.Sqrt(-2*math.Log(u[2*i])) * math.Cos(twoPi*u[2*i+1])
			}
		}
		iv += halfLV
	}
}
