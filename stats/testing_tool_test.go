package stats_test

import "math"

func almostEqual(a, b, tol float64) bool {
	return math.Abs(a-b) < tol
}

func floatAlmostEqual(a, b float64, eps float64) bool {
	return math.Abs(a-b) <= eps
}

func floatSliceAlmostEqual(a, b []float64, eps float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !floatAlmostEqual(a[i], b[i], eps) {
			return false
		}
	}
	return true
}
