package ta

import (
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

func near(a, b float64) bool { return math.Abs(a-b) < 1e-12 }

func toAny(vs []float64) []any {
	out := make([]any, len(vs))
	for i, v := range vs {
		out[i] = v
	}
	return out
}

func nearlyAll(t *testing.T, name string, got *insyra.DataList, want []float64) {
	t.Helper()
	g := floats(t, got)
	if len(g) != len(want) {
		t.Fatalf("%s: got %v, want %v", name, g, want)
	}
	for i := range want {
		if math.Abs(g[i]-want[i]) > 1e-12 {
			t.Fatalf("%s: row index %d: got %v, want %v (all %v)", name, i, g[i], want[i], g)
		}
	}
}
