package insyra

import (
	"math"
	"reflect"
	"testing"
)

func scaleTestTable() *DataTable {
	return NewDataTable(
		NewDataList(1.0, 2.0, 3.0, 4.0).SetName("age"),
		NewDataList(10.0, 20.0, 30.0, 40.0).SetName("income"),
		NewDataList("a", "b", "c", "d").SetName("label"),
	)
}

func floatsOf(t *testing.T, dl *DataList) []float64 {
	t.Helper()
	if dl == nil {
		t.Fatalf("got nil DataList")
	}
	raw := dl.Data()
	out := make([]float64, len(raw))
	for i, v := range raw {
		f, ok := ToFloat64Safe(v)
		if !ok {
			t.Fatalf("value %v at %d is not numeric", v, i)
		}
		out[i] = f
	}
	return out
}

func approx(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func assertApproxSlice(t *testing.T, got []float64, want []float64) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range got {
		if !approx(got[i], want[i]) {
			t.Fatalf("at %d: got %v, want %v (full %v)", i, got[i], want[i], got)
		}
	}
}

// --- StandardScaler ---

func TestStandardScalerFitParamsAndRoundTrip(t *testing.T) {
	dt := scaleTestTable()
	out, sc, err := dt.StandardScale("age")
	if err != nil {
		t.Fatalf("StandardScale: %v", err)
	}
	p := sc.Params()["age"]
	if !approx(p.Mean, 2.5) {
		t.Fatalf("mean = %v, want 2.5", p.Mean)
	}
	// sample stdev of 1,2,3,4 = sqrt(5/3)
	wantStd := math.Sqrt(5.0 / 3.0)
	if !approx(p.Std, wantStd) {
		t.Fatalf("std = %v, want %v", p.Std, wantStd)
	}
	got := floatsOf(t, out.GetColByName("age"))
	var sum float64
	for _, v := range got {
		sum += v
	}
	if !approx(sum/float64(len(got)), 0) {
		t.Fatalf("scaled mean = %v, want ~0", sum/float64(len(got)))
	}
	// untouched columns pass through
	assertApproxSlice(t, floatsOf(t, out.GetColByName("income")), []float64{10, 20, 30, 40})
	// original is unchanged
	assertApproxSlice(t, floatsOf(t, dt.GetColByName("age")), []float64{1, 2, 3, 4})

	back, err := sc.InverseTransform(out)
	if err != nil {
		t.Fatalf("InverseTransform: %v", err)
	}
	assertApproxSlice(t, floatsOf(t, back.GetColByName("age")), []float64{1, 2, 3, 4})
}

func TestStandardScalerConstantColumnNoPanic(t *testing.T) {
	dt := NewDataTable(NewDataList(5.0, 5.0, 5.0).SetName("c"))
	out, sc, err := dt.StandardScale("c")
	if err != nil {
		t.Fatalf("StandardScale constant: %v", err)
	}
	if p := sc.Params()["c"]; !approx(p.Std, 0) {
		t.Fatalf("std = %v, want 0", p.Std)
	}
	assertApproxSlice(t, floatsOf(t, out.GetColByName("c")), []float64{0, 0, 0})
}

// --- MinMaxScaler ---

func TestMinMaxScalerDefaultAndCustomRange(t *testing.T) {
	dt := scaleTestTable()
	out, sc, err := dt.MinMaxScale(0, 1, "age")
	if err != nil {
		t.Fatalf("MinMaxScale: %v", err)
	}
	assertApproxSlice(t, floatsOf(t, out.GetColByName("age")), []float64{0, 1.0 / 3, 2.0 / 3, 1})
	if p := sc.Params()["age"]; !approx(p.Min, 1) || !approx(p.Max, 4) {
		t.Fatalf("min/max = %v/%v", p.Min, p.Max)
	}
	back, err := sc.InverseTransform(out)
	if err != nil {
		t.Fatalf("inverse: %v", err)
	}
	assertApproxSlice(t, floatsOf(t, back.GetColByName("age")), []float64{1, 2, 3, 4})

	out2, _, err := dt.MinMaxScale(-1, 1, "age")
	if err != nil {
		t.Fatalf("MinMaxScale custom: %v", err)
	}
	assertApproxSlice(t, floatsOf(t, out2.GetColByName("age")), []float64{-1, -1.0 / 3, 1.0 / 3, 1})
}

func TestMinMaxScalerConstantColumnOutputsFeatureMin(t *testing.T) {
	dt := NewDataTable(NewDataList(7.0, 7.0, 7.0).SetName("c"))
	out, _, err := dt.MinMaxScale(0, 1, "c")
	if err != nil {
		t.Fatalf("MinMaxScale constant: %v", err)
	}
	assertApproxSlice(t, floatsOf(t, out.GetColByName("c")), []float64{0, 0, 0})

	out2, _, err := NewDataTable(NewDataList(7.0, 7.0).SetName("c")).MinMaxScale(2, 5, "c")
	if err != nil {
		t.Fatalf("MinMaxScale constant range: %v", err)
	}
	assertApproxSlice(t, floatsOf(t, out2.GetColByName("c")), []float64{2, 2})
}

// --- RobustScaler ---

func TestRobustScalerParamsAndRoundTrip(t *testing.T) {
	// 1..5 : median 3, Q1 (p*(n+1)=0.25*6=1.5 -> between idx0,1) = 1.5, Q3 = 4.5, IQR = 3
	dt := NewDataTable(NewDataList(1.0, 2.0, 3.0, 4.0, 5.0).SetName("x"))
	out, sc, err := dt.RobustScale("x")
	if err != nil {
		t.Fatalf("RobustScale: %v", err)
	}
	p := sc.Params()["x"]
	if !approx(p.Median, 3) || !approx(p.Q1, 1.5) || !approx(p.Q3, 4.5) || !approx(p.IQR, 3) {
		t.Fatalf("params = %+v", p)
	}
	assertApproxSlice(t, floatsOf(t, out.GetColByName("x")),
		[]float64{(1 - 3) / 3.0, (2 - 3) / 3.0, 0, (4 - 3) / 3.0, (5 - 3) / 3.0})
	back, err := sc.InverseTransform(out)
	if err != nil {
		t.Fatalf("inverse: %v", err)
	}
	assertApproxSlice(t, floatsOf(t, back.GetColByName("x")), []float64{1, 2, 3, 4, 5})
}

func TestRobustScalerOutlierStabilityAndConstant(t *testing.T) {
	base := []any{1.0, 2.0, 3.0, 4.0, 5.0}
	withOutlier := append(append([]any{}, base...), 1000.0)
	pBase := NewRobustScaler()
	if err := pBase.FitDataList(NewDataList(base...)); err != nil {
		t.Fatalf("fit base: %v", err)
	}
	pOut := NewRobustScaler()
	if err := pOut.FitDataList(NewDataList(withOutlier...)); err != nil {
		t.Fatalf("fit outlier: %v", err)
	}
	// median should remain stable (3 vs 3.5), unlike a mean which would blow up
	if m := pOut.Params()[""].Median; m > 4 {
		t.Fatalf("median moved too far with outlier: %v", m)
	}

	dt := NewDataTable(NewDataList(2.0, 2.0, 2.0).SetName("c"))
	out, _, err := dt.RobustScale("c")
	if err != nil {
		t.Fatalf("RobustScale constant: %v", err)
	}
	assertApproxSlice(t, floatsOf(t, out.GetColByName("c")), []float64{0, 0, 0})
}

// --- MaxAbsScaler ---

func TestMaxAbsScalerSignAndRoundTrip(t *testing.T) {
	dt := NewDataTable(NewDataList(-4.0, -2.0, 0.0, 2.0).SetName("x"))
	out, sc, err := dt.MaxAbsScale("x")
	if err != nil {
		t.Fatalf("MaxAbsScale: %v", err)
	}
	if p := sc.Params()["x"]; !approx(p.MaxAbs, 4) {
		t.Fatalf("maxabs = %v", p.MaxAbs)
	}
	assertApproxSlice(t, floatsOf(t, out.GetColByName("x")), []float64{-1, -0.5, 0, 0.5})
	back, err := sc.InverseTransform(out)
	if err != nil {
		t.Fatalf("inverse: %v", err)
	}
	assertApproxSlice(t, floatsOf(t, back.GetColByName("x")), []float64{-4, -2, 0, 2})
}

func TestMaxAbsScalerAllZeroNoPanic(t *testing.T) {
	dt := NewDataTable(NewDataList(0.0, 0.0, 0.0).SetName("c"))
	out, _, err := dt.MaxAbsScale("c")
	if err != nil {
		t.Fatalf("MaxAbsScale zero: %v", err)
	}
	assertApproxSlice(t, floatsOf(t, out.GetColByName("c")), []float64{0, 0, 0})
}

// --- DataTable behavior ---

func TestScalerTransformOnlyFittedColsAndPreservesShape(t *testing.T) {
	dt := scaleTestTable()
	dt.SetName("data")
	dt.SetRowNameByIndex(0, "r0")
	dt.SetRowNameByIndex(3, "r3")

	out, _, err := dt.MinMaxScale(0, 1, "age")
	if err != nil {
		t.Fatalf("MinMaxScale: %v", err)
	}
	// column order and names preserved
	if got := out.ColNames(); !reflect.DeepEqual(got, []string{"age", "income", "label"}) {
		t.Fatalf("cols = %v", got)
	}
	// non-fitted numeric column untouched
	assertApproxSlice(t, floatsOf(t, out.GetColByName("income")), []float64{10, 20, 30, 40})
	// non-numeric column passes through
	if got := out.GetColByName("label").Data(); !reflect.DeepEqual(got, []any{"a", "b", "c", "d"}) {
		t.Fatalf("label = %v", got)
	}
	// table name + row names preserved
	if out.GetName() != "data" {
		t.Fatalf("table name = %q", out.GetName())
	}
	if n, ok := out.GetRowNameByIndex(0); !ok || n != "r0" {
		t.Fatalf("row name 0 = %q ok=%v", n, ok)
	}
}

func TestScalerTrainTestUsesTrainParams(t *testing.T) {
	train := NewDataTable(NewDataList(0.0, 10.0).SetName("x")) // min 0, max 10
	test := NewDataTable(NewDataList(5.0, 20.0).SetName("x"))

	sc := NewMinMaxScaler(0, 1)
	if _, err := sc.FitTransform(train, "x"); err != nil {
		t.Fatalf("fit train: %v", err)
	}
	out, err := sc.Transform(test)
	if err != nil {
		t.Fatalf("transform test: %v", err)
	}
	// test scaled with TRAIN min/max (0..10), so 5 -> 0.5, 20 -> 2.0 (not re-fit to 1.0)
	assertApproxSlice(t, floatsOf(t, out.GetColByName("x")), []float64{0.5, 2.0})
}

func TestScalerTransformMissingFittedColumnErrors(t *testing.T) {
	dt := scaleTestTable()
	sc := NewStandardScaler()
	if err := sc.Fit(dt, "age"); err != nil {
		t.Fatalf("fit: %v", err)
	}
	other := NewDataTable(NewDataList(1.0, 2.0).SetName("income"))
	if _, err := sc.Transform(other); err == nil {
		t.Fatalf("expected error for missing fitted column")
	}
}

func TestScalerNonNumericColumnErrors(t *testing.T) {
	dt := scaleTestTable()
	sc := NewStandardScaler()
	if err := sc.Fit(dt, "label"); err == nil {
		t.Fatalf("expected error fitting non-numeric column")
	}
}

func TestScalerNaNAndNilPreserved(t *testing.T) {
	dt := NewDataTable(NewDataList(1.0, math.NaN(), nil, 3.0).SetName("x"))
	out, sc, err := dt.MinMaxScale(0, 1, "x")
	if err != nil {
		t.Fatalf("MinMaxScale: %v", err)
	}
	// fit ignores NaN/nil -> min 1, max 3
	if p := sc.Params()["x"]; !approx(p.Min, 1) || !approx(p.Max, 3) {
		t.Fatalf("params = %+v", p)
	}
	data := out.GetColByName("x").Data()
	if f, _ := ToFloat64Safe(data[0]); !approx(f, 0) {
		t.Fatalf("data[0] = %v, want 0", data[0])
	}
	if f, ok := data[1].(float64); !ok || !math.IsNaN(f) {
		t.Fatalf("data[1] = %v, want NaN preserved", data[1])
	}
	if data[2] != nil {
		t.Fatalf("data[2] = %v, want nil preserved", data[2])
	}
	if f, _ := ToFloat64Safe(data[3]); !approx(f, 1) {
		t.Fatalf("data[3] = %v, want 1", data[3])
	}
}

func TestScalerExcelStyleColumnRef(t *testing.T) {
	dt := scaleTestTable()
	// "B" -> second column = income
	out, sc, err := dt.MinMaxScale(0, 1, "B")
	if err != nil {
		t.Fatalf("MinMaxScale by index: %v", err)
	}
	if _, ok := sc.Params()["income"]; !ok {
		t.Fatalf("params keyed by %v, want income", sc.Params())
	}
	assertApproxSlice(t, floatsOf(t, out.GetColByName("income")), []float64{0, 1.0 / 3, 2.0 / 3, 1})
	// age untouched
	assertApproxSlice(t, floatsOf(t, out.GetColByName("age")), []float64{1, 2, 3, 4})
}

func TestScalerFitRequiresColumns(t *testing.T) {
	dt := scaleTestTable()
	sc := NewStandardScaler()
	if err := sc.Fit(dt); err == nil {
		t.Fatalf("expected error when no columns given")
	}
}

func TestScalerTransformBeforeFitErrors(t *testing.T) {
	dt := scaleTestTable()
	sc := NewStandardScaler()
	if _, err := sc.Transform(dt); err == nil {
		t.Fatalf("expected error transforming before fit")
	}
}

// --- DataList API ---

func TestDataListScalerFitTransformReturnsNewList(t *testing.T) {
	dl := NewDataList(1.0, 2.0, 3.0, 4.0).SetName("x")
	sc := NewStandardScaler()
	out, err := sc.FitTransformDataList(dl)
	if err != nil {
		t.Fatalf("FitTransformDataList: %v", err)
	}
	// original unchanged
	assertApproxSlice(t, floatsOf(t, dl), []float64{1, 2, 3, 4})
	// out has zero mean
	var sum float64
	for _, v := range floatsOf(t, out) {
		sum += v
	}
	if !approx(sum/4, 0) {
		t.Fatalf("scaled mean = %v", sum/4)
	}
	if out.GetName() != "x" {
		t.Fatalf("name = %q, want x", out.GetName())
	}
	back, err := sc.InverseTransformDataList(out)
	if err != nil {
		t.Fatalf("InverseTransformDataList: %v", err)
	}
	assertApproxSlice(t, floatsOf(t, back), []float64{1, 2, 3, 4})
}

func TestKindReporting(t *testing.T) {
	cases := map[string]Scaler{
		"standard": NewStandardScaler(),
		"minmax":   NewDefaultMinMaxScaler(),
		"robust":   NewRobustScaler(),
		"maxabs":   NewMaxAbsScaler(),
	}
	for want, sc := range cases {
		if got := sc.Kind(); got != want {
			t.Fatalf("Kind = %q, want %q", got, want)
		}
	}
}
