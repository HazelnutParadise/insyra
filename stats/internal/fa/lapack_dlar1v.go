// fa/lapack_dlar1v.go
//
// dlar1v — compute the (scaled) r-th column of (LDL^T - λI)^{-1} via
// twisted factorization. Used by dlarrv as the per-eigenvector kernel
// of the MRRR algorithm. Faithful translation of LAPACK 3.12.1
// dlar1v.f.
package fa

import "math"

// dlar1v inputs:
//
//	n, b1, bn  : matrix order and active sub-block (1-based, b1<=bn<=n)
//	lambda     : shift (eigenvalue approximation)
//	d          : diagonal of D (length n)
//	l          : sub-diagonal of L (length n-1)
//	ld         : l[i]*d[i] (length n-1)
//	lld        : l[i]*l[i]*d[i] (length n-1)
//	pivmin     : minimum pivot magnitude
//	gaptol     : tolerance for declaring eigenvector entries negligible
//	z          : on input zeroed; on output the (scaled) r-th column of inverse
//	wantnc     : whether to compute negcnt
//	rIn        : on input twist index (0 to autoselect, else 1..n)
//	work       : workspace, length 4*n
//
// Outputs (returned, in order):
//
//	negcnt, ztz, mingma, rOut, isuppz1, isuppz2, nrminv, resid, rqcorr
func dlar1v(
	n, b1, bn int,
	lambda float64,
	d, l, ld, lld []float64,
	pivmin, gaptol float64,
	z []float64,
	wantnc bool,
	rIn int,
	work []float64,
) (negcnt int, ztz, mingma float64, rOut, isuppz1, isuppz2 int, nrminv, resid, rqcorr float64) {

	const eps = 2.2204460492503131e-16

	var r1, r2 int
	if rIn == 0 {
		r1 = b1
		r2 = bn
	} else {
		r1 = rIn
		r2 = rIn
	}

	// Workspace partition (0-based offsets):
	//   indlpl = 0      : LPLUS, length N
	//   indumn = N      : UMINUS, length N
	//   inds   = 2N     : S sequence, length N
	//   indp   = 3N     : P sequence, length N
	const indlpl = 0
	indumn := n
	inds := 2 * n
	indp := 3 * n

	if b1 == 1 {
		work[inds+0] = 0
	} else {
		work[inds+b1-1] = lld[b1-2]
	}

	// Stationary qd transform up to R2.
	sawnan1 := false
	neg1 := 0
	s := work[inds+b1-1] - lambda
	for i := b1; i <= r1-1; i++ {
		dPlus := d[i-1] + s
		work[indlpl+i-1] = ld[i-1] / dPlus
		if dPlus < 0 {
			neg1++
		}
		// Fortran WORK(INDS+I) is 1-based; with inds=2*N as 0-based
		// offset, this is work[inds + i].
		work[inds+i] = s * work[indlpl+i-1] * l[i-1]
		s = work[inds+i] - lambda
	}
	if math.IsNaN(s) {
		sawnan1 = true
	}
	if !sawnan1 {
		for i := r1; i <= r2-1; i++ {
			dPlus := d[i-1] + s
			work[indlpl+i-1] = ld[i-1] / dPlus
			work[inds+i] = s * work[indlpl+i-1] * l[i-1]
			s = work[inds+i] - lambda
		}
		if math.IsNaN(s) {
			sawnan1 = true
		}
	}
	if sawnan1 {
		// Slower NaN-tolerant path.
		neg1 = 0
		s = work[inds+b1-1] - lambda
		for i := b1; i <= r1-1; i++ {
			dPlus := d[i-1] + s
			if math.Abs(dPlus) < pivmin {
				dPlus = -pivmin
			}
			work[indlpl+i-1] = ld[i-1] / dPlus
			if dPlus < 0 {
				neg1++
			}
			work[inds+i] = s * work[indlpl+i-1] * l[i-1]
			if work[indlpl+i-1] == 0 {
				work[inds+i] = lld[i-1]
			}
			s = work[inds+i] - lambda
		}
		for i := r1; i <= r2-1; i++ {
			dPlus := d[i-1] + s
			if math.Abs(dPlus) < pivmin {
				dPlus = -pivmin
			}
			work[indlpl+i-1] = ld[i-1] / dPlus
			work[inds+i] = s * work[indlpl+i-1] * l[i-1]
			if work[indlpl+i-1] == 0 {
				work[inds+i] = lld[i-1]
			}
			s = work[inds+i] - lambda
		}
	}

	// Progressive qd transform from BN down to R1.
	sawnan2 := false
	neg2 := 0
	work[indp+bn-1] = d[bn-1] - lambda
	for i := bn - 1; i >= r1; i-- {
		dMinus := lld[i-1] + work[indp+i]
		tmp := d[i-1] / dMinus
		if dMinus < 0 {
			neg2++
		}
		work[indumn+i-1] = l[i-1] * tmp
		work[indp+i-1] = work[indp+i]*tmp - lambda
	}
	if math.IsNaN(work[indp+r1-1]) {
		sawnan2 = true
	}
	if sawnan2 {
		neg2 = 0
		for i := bn - 1; i >= r1; i-- {
			dMinus := lld[i-1] + work[indp+i]
			if math.Abs(dMinus) < pivmin {
				dMinus = -pivmin
			}
			tmp := d[i-1] / dMinus
			if dMinus < 0 {
				neg2++
			}
			work[indumn+i-1] = l[i-1] * tmp
			work[indp+i-1] = work[indp+i]*tmp - lambda
			if tmp == 0 {
				work[indp+i-1] = d[i-1] - lambda
			}
		}
	}

	// Find the largest-magnitude diagonal of the inverse over [R1,R2].
	mingma = work[inds+r1-1] + work[indp+r1-1]
	if mingma < 0 {
		neg1++
	}
	if wantnc {
		negcnt = neg1 + neg2
	} else {
		negcnt = -1
	}
	if math.Abs(mingma) == 0 {
		mingma = eps * work[inds+r1-1]
	}
	r := r1
	for i := r1; i <= r2-1; i++ {
		tmp := work[inds+i] + work[indp+i]
		if tmp == 0 {
			tmp = eps * work[inds+i]
		}
		if math.Abs(tmp) <= math.Abs(mingma) {
			mingma = tmp
			r = i + 1
		}
	}

	// Compute the FP vector by solving N^T v = e_r.
	isuppz1 = b1
	isuppz2 = bn
	z[r-1] = 1.0
	ztz = 1.0

	// Upward sweep from R-1 down to B1.
	if !sawnan1 && !sawnan2 {
		earlyExit := false
		for i := r - 1; i >= b1; i-- {
			z[i-1] = -(work[indlpl+i-1] * z[i])
			if (math.Abs(z[i-1])+math.Abs(z[i]))*math.Abs(ld[i-1]) < gaptol {
				z[i-1] = 0
				isuppz1 = i + 1
				earlyExit = true
				break
			}
			ztz += z[i-1] * z[i-1]
		}
		_ = earlyExit
	} else {
		for i := r - 1; i >= b1; i-- {
			if z[i] == 0 {
				z[i-1] = -(ld[i] / ld[i-1]) * z[i+1]
			} else {
				z[i-1] = -(work[indlpl+i-1] * z[i])
			}
			if (math.Abs(z[i-1])+math.Abs(z[i]))*math.Abs(ld[i-1]) < gaptol {
				z[i-1] = 0
				isuppz1 = i + 1
				break
			}
			ztz += z[i-1] * z[i-1]
		}
	}

	// Downward sweep from R to BN-1.
	if !sawnan1 && !sawnan2 {
		for i := r; i <= bn-1; i++ {
			z[i] = -(work[indumn+i-1] * z[i-1])
			if (math.Abs(z[i-1])+math.Abs(z[i]))*math.Abs(ld[i-1]) < gaptol {
				z[i] = 0
				isuppz2 = i
				break
			}
			ztz += z[i] * z[i]
		}
	} else {
		for i := r; i <= bn-1; i++ {
			if z[i-1] == 0 {
				z[i] = -(ld[i-2] / ld[i-1]) * z[i-2]
			} else {
				z[i] = -(work[indumn+i-1] * z[i-1])
			}
			if (math.Abs(z[i-1])+math.Abs(z[i]))*math.Abs(ld[i-1]) < gaptol {
				z[i] = 0
				isuppz2 = i
				break
			}
			ztz += z[i] * z[i]
		}
	}

	// Convergence-test quantities.
	tmp := 1.0 / ztz
	nrminv = math.Sqrt(tmp)
	resid = math.Abs(mingma) * nrminv
	rqcorr = mingma * tmp
	rOut = r
	return
}
