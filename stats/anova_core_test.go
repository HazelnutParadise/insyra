package stats

import (
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

func TestOneWayANOVACoreMatchesPublic(t *testing.T) {
	g1 := insyra.NewDataList([]float64{10, 12, 9, 11})
	g2 := insyra.NewDataList([]float64{20, 19, 21, 22})
	g3 := insyra.NewDataList([]float64{30, 29, 28, 32})

	public := OneWayANOVA(g1, g2, g3)
	if public == nil {
		t.Fatal("OneWayANOVA returned nil")
	}

	values := []float64{10, 12, 9, 11, 20, 19, 21, 22, 30, 29, 28, 32}
	labels := []int{0, 0, 0, 0, 1, 1, 1, 1, 2, 2, 2, 2}
	core := oneWayANOVAFromSlices(values, labels, 3)
	if core == nil {
		t.Fatal("oneWayANOVAFromSlices returned nil")
	}

	if !coreAlmostEqual(public.Factor.SumOfSquares, core.SSB, 1e-12) {
		t.Fatalf("SSB mismatch: public=%v core=%v", public.Factor.SumOfSquares, core.SSB)
	}
	if !coreAlmostEqual(public.Within.SumOfSquares, core.SSW, 1e-12) {
		t.Fatalf("SSW mismatch: public=%v core=%v", public.Within.SumOfSquares, core.SSW)
	}
	if public.Factor.DF != core.DFB || public.Within.DF != core.DFW {
		t.Fatalf("DF mismatch: public=(%d,%d) core=(%d,%d)", public.Factor.DF, public.Within.DF, core.DFB, core.DFW)
	}
	if !coreAlmostEqual(public.Factor.F, core.F, 1e-12) {
		t.Fatalf("F mismatch: public=%v core=%v", public.Factor.F, core.F)
	}
	if !coreAlmostEqual(public.Factor.P, core.P, 1e-12) {
		t.Fatalf("P mismatch: public=%v core=%v", public.Factor.P, core.P)
	}
	if !coreAlmostEqual(public.Factor.EtaSquared, core.Eta, 1e-12) {
		t.Fatalf("Eta mismatch: public=%v core=%v", public.Factor.EtaSquared, core.Eta)
	}
}

func TestLeveneUsesSharedOneWayCore(t *testing.T) {
	groups := []insyra.IDataList{
		insyra.NewDataList([]float64{10, 12, 9, 11}),
		insyra.NewDataList([]float64{20, 19, 21, 22}),
		insyra.NewDataList([]float64{30, 29, 28, 32}),
	}

	levene := LeveneTest(groups)
	if levene == nil {
		t.Fatal("LeveneTest returned nil")
	}

	allDiffs := make([]float64, 0)
	labels := make([]int, 0)
	for i, group := range groups {
		median := group.Median()
		for _, v := range group.Data() {
			x, ok := insyra.ToFloat64Safe(v)
			if !ok {
				continue
			}
			allDiffs = append(allDiffs, math.Abs(x-median))
			labels = append(labels, i)
		}
	}

	core := oneWayANOVAFromSlices(allDiffs, labels, len(groups))
	if core == nil {
		t.Fatal("oneWayANOVAFromSlices returned nil for Levene data")
	}

	if !coreAlmostEqual(levene.Statistic, core.F, 1e-12) {
		t.Fatalf("Levene F mismatch: got %v want %v", levene.Statistic, core.F)
	}
	if !coreAlmostEqual(levene.PValue, core.P, 1e-12) {
		t.Fatalf("Levene P mismatch: got %v want %v", levene.PValue, core.P)
	}
	if levene.DF == nil || !coreAlmostEqual(*levene.DF, float64(core.DFB), 1e-12) {
		t.Fatalf("Levene DF1 mismatch: got %v want %v", levene.DF, core.DFB)
	}
	if !coreAlmostEqual(levene.DF2, float64(core.DFW), 1e-12) {
		t.Fatalf("Levene DF2 mismatch: got %v want %v", levene.DF2, core.DFW)
	}
}

func coreAlmostEqual(a, b, tol float64) bool {
	return math.Abs(a-b) <= tol
}
