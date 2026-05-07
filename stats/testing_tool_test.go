package stats_test

import "math"

// floatAlmostEqual is kept for the legacy correlation_ci_test.go which
// has its own (very tight, 1e-12) tolerance scheme; the rest of the
// stats test suite uses per-batch helpers (mClose, cClose, etc.) that
// add NaN/Inf handling.
func floatAlmostEqual(a, b float64, eps float64) bool {
	return math.Abs(a-b) <= eps
}
