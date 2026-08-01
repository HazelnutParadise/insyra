package insyra

import (
	"math"
	"testing"
)

// approxEqual returns true when a and b are within tol of each other or both
// NaN. nil pointers are considered unequal to anything but nil.
func approxEqual(a, b any, tol float64) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	af, oka := ToFloat64Safe(a)
	bf, okb := ToFloat64Safe(b)
	if !oka || !okb {
		return a == b
	}
	if math.IsNaN(af) && math.IsNaN(bf) {
		return true
	}
	return math.Abs(af-bf) <= tol
}

func sliceEqualApprox(t *testing.T, got, want []any, tol float64) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("length mismatch: got %d (%v), want %d (%v)", len(got), got, len(want), want)
	}
	for i := range got {
		if !approxEqual(got[i], want[i], tol) {
			t.Errorf("index %d: got %v (%T), want %v (%T)", i, got[i], got[i], want[i], want[i])
		}
	}
}

// =============================================================================
// Shift
// =============================================================================

func TestDataList_Shift_LagPositive(t *testing.T) {
	dl := NewDataList(10.0, 20.0, 30.0, 40.0, 50.0)
	got := dl.Shift(1).Data()
	want := []any{nil, 10.0, 20.0, 30.0, 40.0}
	sliceEqualApprox(t, got, want, 0)
}

func TestDataList_Shift_LeadNegative(t *testing.T) {
	dl := NewDataList(10.0, 20.0, 30.0, 40.0, 50.0)
	got := dl.Shift(-2).Data()
	want := []any{30.0, 40.0, 50.0, nil, nil}
	sliceEqualApprox(t, got, want, 0)
}

func TestDataList_Shift_Zero(t *testing.T) {
	dl := NewDataList(1, 2, 3)
	got := dl.Shift(0).Data()
	want := []any{1, 2, 3}
	sliceEqualApprox(t, got, want, 0)
}

func TestDataList_Shift_BeyondLength(t *testing.T) {
	dl := NewDataList(1, 2, 3)
	got := dl.Shift(10).Data()
	want := []any{nil, nil, nil}
	sliceEqualApprox(t, got, want, 0)
}

func TestDataList_Shift_FillValue(t *testing.T) {
	dl := NewDataList(1, 2, 3, 4)
	got := dl.Shift(2, 0).Data()
	want := []any{0, 0, 1, 2}
	sliceEqualApprox(t, got, want, 0)
}

func TestDataList_Shift_NonNumeric(t *testing.T) {
	dl := NewDataList("a", "b", "c")
	got := dl.Shift(1).Data()
	want := []any{nil, "a", "b"}
	sliceEqualApprox(t, got, want, 0)
}

func TestDataList_Shift_Empty(t *testing.T) {
	dl := NewDataList()
	got := dl.Shift(2).Data()
	if len(got) != 0 {
		t.Fatalf("expected empty result, got %v", got)
	}
}

// =============================================================================
// Diff
// =============================================================================

func TestDataList_Diff_OneStep(t *testing.T) {
	dl := NewDataList(10.0, 13.0, 18.0, 17.0, 25.0)
	got := dl.Diff(1).Data()
	want := []any{nil, 3.0, 5.0, -1.0, 8.0}
	sliceEqualApprox(t, got, want, 1e-9)
}

func TestDataList_Diff_TwoStep(t *testing.T) {
	dl := NewDataList(10.0, 13.0, 18.0, 17.0, 25.0)
	got := dl.Diff(2).Data()
	want := []any{nil, nil, 8.0, 4.0, 7.0}
	sliceEqualApprox(t, got, want, 1e-9)
}

func TestDataList_Diff_NilInside(t *testing.T) {
	dl := NewDataList(10.0, nil, 18.0, 17.0)
	got := dl.Diff(1).Data()
	want := []any{nil, nil, nil, -1.0}
	sliceEqualApprox(t, got, want, 1e-9)
}

func TestDataList_Diff_BadPeriods(t *testing.T) {
	dl := NewDataList(1.0, 2.0, 3.0)
	if got := dl.Diff(0); got != nil {
		t.Errorf("expected nil for periods=0, got %v", got)
	}
	if got := dl.Diff(-1); got != nil {
		t.Errorf("expected nil for periods=-1, got %v", got)
	}
}

// =============================================================================
// PctChange
// =============================================================================

func TestDataList_PctChange_Basic(t *testing.T) {
	dl := NewDataList(100.0, 110.0, 99.0, 99.0)
	got := dl.PctChange(1).Data()
	want := []any{nil, 0.1, -0.1, 0.0}
	sliceEqualApprox(t, got, want, 1e-9)
}

func TestDataList_PctChange_DivByZero(t *testing.T) {
	dl := NewDataList(0.0, 5.0, 10.0)
	got := dl.PctChange(1).Data()
	want := []any{nil, nil, 1.0} // (5-0)/0=nil, (10-5)/5=1
	sliceEqualApprox(t, got, want, 1e-9)
}

// =============================================================================
// CumSum / CumProd / CumMax / CumMin
// =============================================================================

func TestDataList_CumSum(t *testing.T) {
	dl := NewDataList(10.0, 20.0, 30.0, 40.0)
	got := dl.CumSum().Data()
	want := []any{10.0, 30.0, 60.0, 100.0}
	sliceEqualApprox(t, got, want, 1e-9)
}

func TestDataList_CumSum_WithNil(t *testing.T) {
	dl := NewDataList(1.0, nil, 3.0, 4.0)
	got := dl.CumSum().Data()
	want := []any{1.0, nil, 4.0, 8.0}
	sliceEqualApprox(t, got, want, 1e-9)
}

func TestDataList_CumProd(t *testing.T) {
	dl := NewDataList(2.0, 3.0, 4.0)
	got := dl.CumProd().Data()
	want := []any{2.0, 6.0, 24.0}
	sliceEqualApprox(t, got, want, 1e-9)
}

func TestDataList_CumMax(t *testing.T) {
	dl := NewDataList(3.0, 1.0, 4.0, 1.0, 5.0, 9.0, 2.0)
	got := dl.CumMax().Data()
	want := []any{3.0, 3.0, 4.0, 4.0, 5.0, 9.0, 9.0}
	sliceEqualApprox(t, got, want, 1e-9)
}

func TestDataList_CumMin(t *testing.T) {
	dl := NewDataList(3.0, 1.0, 4.0, 1.0, 5.0, 9.0, 2.0)
	got := dl.CumMin().Data()
	want := []any{3.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0}
	sliceEqualApprox(t, got, want, 1e-9)
}

func TestDataList_CumMax_LeadingNil(t *testing.T) {
	dl := NewDataList(nil, 3.0, 1.0)
	got := dl.CumMax().Data()
	want := []any{nil, 3.0, 3.0}
	sliceEqualApprox(t, got, want, 1e-9)
}

// =============================================================================
// Rolling
// =============================================================================

func TestDataList_Rolling_Mean_Basic(t *testing.T) {
	dl := NewDataList(1.0, 2.0, 3.0, 4.0, 5.0)
	got := dl.Rolling(RollingOptions{Window: 3}).Mean().Data()
	want := []any{nil, nil, 2.0, 3.0, 4.0}
	sliceEqualApprox(t, got, want, 1e-9)
}

func TestDataList_Rolling_Mean_MinObs(t *testing.T) {
	dl := NewDataList(1.0, 2.0, 3.0, 4.0, 5.0)
	got := dl.Rolling(RollingOptions{Window: 3, MinObs: 1}).Mean().Data()
	want := []any{1.0, 1.5, 2.0, 3.0, 4.0}
	sliceEqualApprox(t, got, want, 1e-9)
}

func TestDataList_Rolling_Sum(t *testing.T) {
	dl := NewDataList(1.0, 2.0, 3.0, 4.0, 5.0)
	got := dl.Rolling(RollingOptions{Window: 2}).Sum().Data()
	want := []any{nil, 3.0, 5.0, 7.0, 9.0}
	sliceEqualApprox(t, got, want, 1e-9)
}

func TestDataList_Rolling_Min(t *testing.T) {
	dl := NewDataList(5.0, 1.0, 4.0, 2.0, 3.0)
	got := dl.Rolling(RollingOptions{Window: 3}).Min().Data()
	want := []any{nil, nil, 1.0, 1.0, 2.0}
	sliceEqualApprox(t, got, want, 1e-9)
}

func TestDataList_Rolling_Max(t *testing.T) {
	dl := NewDataList(5.0, 1.0, 4.0, 2.0, 3.0)
	got := dl.Rolling(RollingOptions{Window: 3}).Max().Data()
	want := []any{nil, nil, 5.0, 4.0, 4.0}
	sliceEqualApprox(t, got, want, 1e-9)
}

func TestDataList_Rolling_Median(t *testing.T) {
	dl := NewDataList(1.0, 3.0, 2.0, 4.0, 5.0)
	got := dl.Rolling(RollingOptions{Window: 3}).Median().Data()
	want := []any{nil, nil, 2.0, 3.0, 4.0}
	sliceEqualApprox(t, got, want, 1e-9)
}

func TestDataList_Rolling_Std(t *testing.T) {
	// Sample std of [1,2,3] = sqrt(1) = 1
	dl := NewDataList(1.0, 2.0, 3.0, 4.0, 5.0)
	got := dl.Rolling(RollingOptions{Window: 3}).Std().Data()
	want := []any{nil, nil, 1.0, 1.0, 1.0}
	sliceEqualApprox(t, got, want, 1e-9)
}

func TestDataList_Rolling_Var(t *testing.T) {
	dl := NewDataList(1.0, 2.0, 3.0, 4.0, 5.0)
	got := dl.Rolling(RollingOptions{Window: 3}).Var().Data()
	want := []any{nil, nil, 1.0, 1.0, 1.0}
	sliceEqualApprox(t, got, want, 1e-9)
}

func TestDataList_Rolling_WithNil(t *testing.T) {
	dl := NewDataList(1.0, nil, 3.0, 4.0, 5.0)
	got := dl.Rolling(RollingOptions{Window: 3, MinObs: 2}).Mean().Data()
	// position 2: window=[1, nil, 3], valid=[1,3] count=2 >= 2, mean=2
	// position 3: window=[nil, 3, 4], valid=[3,4] count=2, mean=3.5
	// position 4: window=[3, 4, 5], mean=4
	want := []any{nil, nil, 2.0, 3.5, 4.0}
	sliceEqualApprox(t, got, want, 1e-9)
}

func TestDataList_Rolling_WeightedMean(t *testing.T) {
	dl := NewDataList(1.0, 2.0, 3.0, 4.0)
	got := dl.Rolling(RollingOptions{Window: 2, Weights: []float64{1, 3}}).Mean().Data()
	// position 1: (1*1 + 2*3) / 4 = 7/4 = 1.75
	// position 2: (2*1 + 3*3) / 4 = 11/4 = 2.75
	// position 3: (3*1 + 4*3) / 4 = 15/4 = 3.75
	want := []any{nil, 1.75, 2.75, 3.75}
	sliceEqualApprox(t, got, want, 1e-9)
}

func TestDataList_Rolling_Center(t *testing.T) {
	dl := NewDataList(1.0, 2.0, 3.0, 4.0, 5.0)
	got := dl.Rolling(RollingOptions{Window: 3, Center: true}).Mean().Data()
	// center=true window=3: covers [i-1, i, i+1]
	// position 0: [-1,0,1] clipped to [0,1] -> count=2 < window=3 -> nil
	// position 1: [0,1,2] -> 2
	// position 2: [1,2,3] -> 3
	// position 3: [2,3,4] -> 4
	// position 4: [3,4,5] clipped to [3,4] -> count=2 < 3 -> nil
	want := []any{nil, 2.0, 3.0, 4.0, nil}
	sliceEqualApprox(t, got, want, 1e-9)
}

func TestDataList_Rolling_BadWindow(t *testing.T) {
	dl := NewDataList(1.0, 2.0)
	got := dl.Rolling(RollingOptions{Window: 0}).Mean()
	if got == nil || len(got.Data()) != 0 {
		t.Errorf("expected empty result on Window=0, got %v", got.Data())
	}
}

func TestDataList_Rolling_Apply(t *testing.T) {
	dl := NewDataList(10.0, 20.0, 30.0, 40.0)
	// Custom reducer: sum of squares
	got := dl.Rolling(RollingOptions{Window: 2}).Apply(func(w []any) any {
		var s float64
		for _, v := range w {
			f, _ := ToFloat64Safe(v)
			s += f * f
		}
		return s
	}).Data()
	// pos 1: 10²+20² = 500
	// pos 2: 20²+30² = 1300
	// pos 3: 30²+40² = 2500
	want := []any{nil, 500.0, 1300.0, 2500.0}
	sliceEqualApprox(t, got, want, 1e-9)
}

func TestDataList_Rolling_Corr(t *testing.T) {
	x := NewDataList(1.0, 2.0, 3.0, 4.0, 5.0)
	y := NewDataList(2.0, 4.0, 6.0, 8.0, 10.0)
	got := x.Rolling(RollingOptions{Window: 3}).Corr(y).Data()
	// Perfect positive linear correlation in every window -> 1.0
	want := []any{nil, nil, 1.0, 1.0, 1.0}
	sliceEqualApprox(t, got, want, 1e-9)
}

// =============================================================================
// Expanding
// =============================================================================

func TestDataList_Expanding_Mean(t *testing.T) {
	dl := NewDataList(1.0, 2.0, 3.0, 4.0)
	got := dl.Expanding(1).Mean().Data()
	want := []any{1.0, 1.5, 2.0, 2.5}
	sliceEqualApprox(t, got, want, 1e-9)
}

func TestDataList_Expanding_MinObs(t *testing.T) {
	dl := NewDataList(1.0, 2.0, 3.0, 4.0)
	got := dl.Expanding(3).Mean().Data()
	want := []any{nil, nil, 2.0, 2.5}
	sliceEqualApprox(t, got, want, 1e-9)
}

func TestDataList_Expanding_Sum(t *testing.T) {
	dl := NewDataList(1.0, 2.0, 3.0, 4.0)
	got := dl.Expanding(1).Sum().Data()
	want := []any{1.0, 3.0, 6.0, 10.0}
	sliceEqualApprox(t, got, want, 1e-9)
}

func TestDataList_Expanding_Min(t *testing.T) {
	dl := NewDataList(3.0, 1.0, 4.0, 1.0, 5.0)
	got := dl.Expanding(1).Min().Data()
	want := []any{3.0, 1.0, 1.0, 1.0, 1.0}
	sliceEqualApprox(t, got, want, 1e-9)
}

func TestDataList_Expanding_Max(t *testing.T) {
	dl := NewDataList(3.0, 1.0, 4.0, 1.0, 5.0)
	got := dl.Expanding(1).Max().Data()
	want := []any{3.0, 3.0, 4.0, 4.0, 5.0}
	sliceEqualApprox(t, got, want, 1e-9)
}

func TestDataList_Expanding_Std(t *testing.T) {
	dl := NewDataList(1.0, 2.0, 3.0, 4.0, 5.0)
	got := dl.Expanding(1).Std().Data()
	// [1] -> nil (need >=2)
	// [1,2] -> sample std = sqrt(0.5) = 0.7071067811865476
	// [1,2,3] -> sample std = 1
	// [1,2,3,4] -> sample std = sqrt(5/3) ≈ 1.2909944487358056
	// [1,2,3,4,5] -> sample std = sqrt(2.5) ≈ 1.5811388300841898
	want := []any{nil, math.Sqrt(0.5), 1.0, math.Sqrt(5.0 / 3), math.Sqrt(2.5)}
	sliceEqualApprox(t, got, want, 1e-9)
}

// =============================================================================
// Name preservation
// =============================================================================

func TestDataList_Shift_PreservesName(t *testing.T) {
	dl := NewDataList(1.0, 2.0, 3.0)
	dl.SetName("price")
	out := dl.Shift(1)
	if out == nil {
		t.Fatal("Shift returned nil")
	}
	// Access name via direct field — not part of public API but exposed for tests.
	if got := dataListName(out); got != "price" {
		t.Errorf("expected name 'price', got %q", got)
	}
}

// dataListName is a tiny test helper exposing the unexported name field.
func dataListName(dl *DataList) string {
	var n string
	dl.AtomicDo(func(d *DataList) {
		n = d.name
	})
	return n
}
