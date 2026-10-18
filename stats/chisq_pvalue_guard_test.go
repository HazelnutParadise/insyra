package stats

import (
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

func TestChiSquaredPValueNeverPanics(t *testing.T) {
	if got := chiSquaredPValue(-1e-16, 1); got != 1 {
		t.Fatalf("negative statistic: p = %v, want 1", got)
	}
	if got := chiSquaredPValue(0, 1); got != 1 {
		t.Fatalf("zero statistic: p = %v, want 1", got)
	}
	for _, c := range [][2]float64{{1, 0}, {1, -1}, {math.NaN(), 1}, {1, math.NaN()}} {
		if got := chiSquaredPValue(c[0], c[1]); !math.IsNaN(got) {
			t.Fatalf("chiSquaredPValue(%v, %v) = %v, want NaN", c[0], c[1], got)
		}
	}
	if got := chiSquaredPValue(3.841458820694124, 1); math.Abs(got-0.05) > 1e-9 {
		t.Fatalf("ordinary statistic moved: p = %v", got)
	}
}

func zeros(n int) *insyra.DataList {
	d := insyra.NewDataList()
	for range n {
		d.Append(0)
	}
	return d
}

func TestAllEqualDataDoesNotPanic(t *testing.T) {
	r, err := KruskalWallis([]insyra.IDataList{zeros(24), zeros(4), zeros(27), zeros(11)})
	if err != nil {
		t.Fatal(err)
	}
	if r.PValue != 1 {
		t.Fatalf("Kruskal-Wallis on all-equal data: p = %v, want 1", r.PValue)
	}
	subjects := make([]insyra.IDataList, 21)
	for i := range subjects {
		d := insyra.NewDataList()
		for range 7 {
			d.Append(0)
		}
		subjects[i] = d
	}
	f, err := FriedmanTest(subjects)
	if err != nil {
		t.Fatal(err)
	}
	if f.PValue != 1 {
		t.Fatalf("Friedman on all-equal data: p = %v, want 1", f.PValue)
	}
}

func TestBartlettEqualVariancesDoesNotPanic(t *testing.T) {
	g := []float64{-0.32329881667362437, -0.319580921043235, 0.9093452052941211, 0.9788142136474671}
	a, b := insyra.NewDataList(), insyra.NewDataList()
	for _, v := range g {
		a.Append(v)
		b.Append(v)
	}
	r, err := BartlettTest([]insyra.IDataList{a, b})
	if err != nil {
		t.Fatal(err)
	}
	if r.PValue != 1 {
		t.Fatalf("Bartlett on identical groups: p = %v, want 1", r.PValue)
	}
}
