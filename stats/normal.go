package stats

import (
	"fmt"
	"math"
)

// NormCDF returns Φ(x), the cumulative distribution function of the
// standard normal distribution N(0, 1), evaluated at x:
//
//	Φ(x) = P(Z ≤ x),  Z ~ N(0, 1)
//
// It is defined for every real x (x = -Inf → 0, x = +Inf → 1, NaN → NaN),
// so it never fails and does not return an error.
//
// This is the public, standalone counterpart of the distribution helpers
// used internally by the z-test and confidence-interval machinery.
func NormCDF(x float64) float64 {
	return zCDF(x)
}

// NormPPF returns Φ⁻¹(p), the inverse CDF — also called the quantile or
// percent-point function — of the standard normal distribution N(0, 1):
//
//	Φ⁻¹(p) = x  such that  Φ(x) = p
//
// p must lie in [0, 1]. The boundaries return the (correct) infinite
// quantiles: p = 0 → -Inf, p = 1 → +Inf. A p outside [0, 1] or a NaN is
// invalid input and returns an error, following the stats package's
// error-first convention for exported functions.
func NormPPF(p float64) (float64, error) {
	if math.IsNaN(p) || p < 0 || p > 1 {
		return math.NaN(), fmt.Errorf("NormPPF: p must be in [0, 1], got %v", p)
	}
	return zQuantile(p), nil
}
