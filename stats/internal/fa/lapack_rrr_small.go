// fa/lapack_rrr_small.go
//
// Small RRR (relatively robust representations) helpers used by the
// dsyevr code path.
//
//   dlarrr  — Determine if relative-accuracy preserving computations
//             are advisable for a tridiagonal matrix.
//   dlarrk  — Compute one eigenvalue of a symmetric tridiagonal matrix
//             by bisection.
//
// Faithful translations of LAPACK 3.12.1 reference Fortran (dlarrr.f /
// dlarrk.f). Both are leaves — no further LAPACK dependencies.
package fa

import "math"

// dlarrr decides whether relative-accuracy preserving computations are
// advisable for the symmetric tridiagonal matrix with diagonal d
// (length n) and off-diagonal e (length n-1).
//
// Returns: 0 = relative-accuracy preserved (RRR mode appropriate);
//          1 = otherwise (the safer absolute-accuracy bisection path).
//
// Mirrors LAPACK dlarrr.f. Only the "scaled diagonal dominance" test is
// implemented (the U/L bidiagonal-conditioning tests are stubs in the
// reference Fortran too — they read "*** MORE TO BE IMPLEMENTED ***").
func dlarrr(n int, d, e []float64) (info int) {
	if n <= 0 {
		return 0
	}
	const relCond = 0.999
	// Default: do NOT use RRR.
	info = 1
	const safmin = lapackSafmin
	const eps = 2.2204460492503131e-16
	smlnum := safmin / eps
	rmin := math.Sqrt(smlnum)

	yesRel := true
	offDig := 0.0
	tmp := math.Sqrt(math.Abs(d[0]))
	if tmp < rmin {
		yesRel = false
	}
	if yesRel {
		for i := 2; i <= n; i++ {
			tmp2 := math.Sqrt(math.Abs(d[i-1]))
			if tmp2 < rmin {
				yesRel = false
				break
			}
			offDig2 := math.Abs(e[i-2]) / (tmp * tmp2)
			if offDig+offDig2 >= relCond {
				yesRel = false
				break
			}
			tmp = tmp2
			offDig = offDig2
		}
	}
	if yesRel {
		return 0
	}
	// (LAPACK reference: more tests are stubbed out here.)
	return 1
}

// dlarrk computes the iw-th eigenvalue of a symmetric tridiagonal matrix
// via bisection on Sturm sequences.
//
// Inputs:
//
//	n       : matrix order
//	iw      : index (1-based) of the eigenvalue to compute
//	gl, gu  : Gershgorin lower / upper bounds on the spectrum
//	d       : diagonal (length n)
//	e2      : squared off-diagonal e[i]^2 (length n-1)
//	pivmin  : minimum allowed pivot magnitude
//	reltol  : relative tolerance for the bisection interval
//
// Returns:
//
//	w       : the computed eigenvalue (mid-point of converged interval)
//	werr    : interval half-width
//	info    : 0 converged, -1 did not converge
//
// Mirrors LAPACK dlarrk.f.
func dlarrk(n, iw int, gl, gu float64, d, e2 []float64,
	pivmin, reltol float64,
) (w, werr float64, info int) {
	if n <= 0 {
		return 0, 0, 0
	}
	const fudge = 2.0
	const eps = 2.2204460492503131e-16

	tnorm := math.Max(math.Abs(gl), math.Abs(gu))
	rtoli := reltol
	atoli := fudge * 2 * pivmin
	itmax := int((math.Log(tnorm+pivmin)-math.Log(pivmin))/math.Log(2)) + 2

	info = -1
	left := gl - fudge*tnorm*eps*float64(n) - fudge*2*pivmin
	right := gu + fudge*tnorm*eps*float64(n) + fudge*2*pivmin
	it := 0

	for {
		// Check if interval converged or maxit reached.
		tmp1 := math.Abs(right - left)
		tmp2 := math.Max(math.Abs(right), math.Abs(left))
		if tmp1 < math.Max(atoli, math.Max(pivmin, rtoli*tmp2)) {
			info = 0
			break
		}
		if it > itmax {
			break
		}
		// Count number of negative pivots at midpoint.
		it++
		mid := 0.5 * (left + right)
		negCnt := 0
		tmp := d[0] - mid
		if math.Abs(tmp) < pivmin {
			tmp = -pivmin
		}
		if tmp <= 0 {
			negCnt++
		}
		for i := 2; i <= n; i++ {
			tmp = d[i-1] - e2[i-2]/tmp - mid
			if math.Abs(tmp) < pivmin {
				tmp = -pivmin
			}
			if tmp <= 0 {
				negCnt++
			}
		}
		if negCnt >= iw {
			right = mid
		} else {
			left = mid
		}
	}
	w = 0.5 * (left + right)
	werr = 0.5 * math.Abs(right-left)
	return
}
