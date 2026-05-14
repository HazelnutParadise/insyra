package isr

import (
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

func buildIsrTable() *dt {
	price := insyra.NewDataList(10.0, 12.0, 11.0, 13.0, 18.0)
	price.SetName("price")
	return UseDT(insyra.NewDataTable(price))
}

func approxEqualSlice(t *testing.T, got []any, want []any, tol float64) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("length mismatch: got %d (%v), want %d (%v)", len(got), got, len(want), want)
	}
	for i := range got {
		if got[i] == nil && want[i] == nil {
			continue
		}
		if got[i] == nil || want[i] == nil {
			t.Errorf("[%d] got %v, want %v", i, got[i], want[i])
			continue
		}
		gf, gok := insyra.ToFloat64Safe(got[i])
		wf, wok := insyra.ToFloat64Safe(want[i])
		if gok && wok && math.Abs(gf-wf) > tol {
			t.Errorf("[%d] got %v, want %v", i, got[i], want[i])
		}
	}
}

func TestIsr_Shift(t *testing.T) {
	d := buildIsrTable()
	got := d.Shift("price", 1).Data()
	approxEqualSlice(t, got, []any{nil, 10.0, 12.0, 11.0, 13.0}, 1e-9)
}

func TestIsr_CumSum(t *testing.T) {
	d := buildIsrTable()
	got := d.CumSum("price").Data()
	approxEqualSlice(t, got, []any{10.0, 22.0, 33.0, 46.0, 64.0}, 1e-9)
}

func TestIsr_Rolling_Mean(t *testing.T) {
	d := buildIsrTable()
	got := d.RollingOn("price", Rolling{Window: 3}).Mean().Data()
	approxEqualSlice(t, got, []any{nil, nil, 11.0, 12.0, 14.0}, 1e-9)
}

func TestIsr_Expanding_Sum(t *testing.T) {
	d := buildIsrTable()
	got := d.ExpandingOn("price", 1).Sum().Data()
	approxEqualSlice(t, got, []any{10.0, 22.0, 33.0, 46.0, 64.0}, 1e-9)
}

func TestIsr_ChainAppend(t *testing.T) {
	d := buildIsrTable()
	d.AppendCols(d.RollingOn("price", Rolling{Window: 2}).Mean().SetName("ma2"))
	if d.NumCols() != 2 {
		t.Fatalf("expected 2 columns, got %d", d.NumCols())
	}
}
