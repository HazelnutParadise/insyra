package ta

import "math"

func near(a, b float64) bool { return math.Abs(a-b) < 1e-12 }

func toAny(vs []float64) []any {
	out := make([]any, len(vs))
	for i, v := range vs {
		out[i] = v
	}
	return out
}
