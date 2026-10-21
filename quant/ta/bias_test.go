package ta

import (
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

func TestBias(t *testing.T) {
	got, err := Bias(insyra.NewDataList(10.0, 12.0, 14.0, 12.0), 2)
	if err != nil {
		t.Fatal(err)
	}
	g := floats(t, got)
	// SMA(2): -, 11, 13, 13 -> 12/11-1, 14/13-1, 12/13-1.
	if !math.IsNaN(g[0]) || !near(g[1], 12.0/11-1) || !near(g[2], 14.0/13-1) || !near(g[3], 12.0/13-1) {
		t.Errorf("got %v", g)
	}
}

func TestBiasZUsesOnlyEarlierBiases(t *testing.T) {
	p := []float64{10, 11, 12, 11, 13, 12, 14, 13}
	z, err := BiasZ(insyra.NewDataList(toAny(p)...), 2, 3)
	if err != nil {
		t.Fatal(err)
	}
	b := biasValues(p, 2)
	g := floats(t, z)
	// First defined at t = n-1+window = 4: the deviation of b[1..3], then b[4].
	hist := b[1:4]
	mean := (hist[0] + hist[1] + hist[2]) / 3
	sd := math.Sqrt(((hist[0]-mean)*(hist[0]-mean) + (hist[1]-mean)*(hist[1]-mean) + (hist[2]-mean)*(hist[2]-mean)) / 2)
	if !math.IsNaN(g[3]) || !near(g[4], b[4]/sd) {
		t.Errorf("got %v, want %v at row index 4", g, b[4]/sd)
	}
	// Changing the future leaves earlier values alone.
	q := append([]float64(nil), p...)
	q[7] = 99
	z2, _ := BiasZ(insyra.NewDataList(toAny(q)...), 2, 3)
	for i := 0; i < 7; i++ {
		a, c := g[i], floats(t, z2)[i]
		if !(a == c || (math.IsNaN(a) && math.IsNaN(c))) {
			t.Errorf("row index %d changed when only the last price did", i)
		}
	}
}
