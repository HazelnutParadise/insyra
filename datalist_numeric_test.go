package insyra

import (
	"math"
	"reflect"
	"strings"
	"testing"
)

func init() {
	SetDefaultConfig()
	Config.SetLogLevel(LogLevelFatal)
}

func TestReplaceLastLeavesTrailingNaN(t *testing.T) {
	dl := NewDataList(5, math.NaN())
	dl.ReplaceLast(5, 0)
	got := dl.Data()
	if got[0] != 0 {
		t.Fatalf("expected first cell replaced, got %v", got)
	}
	if f, ok := got[1].(float64); !ok || !math.IsNaN(f) {
		t.Fatalf("expected trailing NaN untouched, got %v", got)
	}
}

func TestReplaceLastReplacesLastMatch(t *testing.T) {
	dl := NewDataList(5, 1, 5)
	dl.ReplaceLast(5, 0)
	if !reflect.DeepEqual(dl.Data(), []any{5, 1, 0}) {
		t.Fatalf("got %v", dl.Data())
	}
}

func TestReplaceLastNaNTargetsLastNaN(t *testing.T) {
	dl := NewDataList(math.NaN(), 1, math.NaN())
	dl.ReplaceLast(math.NaN(), 0)
	got := dl.Data()
	if f, ok := got[0].(float64); !ok || !math.IsNaN(f) {
		t.Fatalf("first NaN should stay, got %v", got)
	}
	if got[2] != 0 {
		t.Fatalf("last NaN should be replaced, got %v", got)
	}
}

func assertUntouchedWithErr(t *testing.T, name string, dl *DataList, want []any) {
	t.Helper()
	if !reflect.DeepEqual(dl.Data(), want) {
		t.Fatalf("%s: data was modified: %v", name, dl.Data())
	}
	if dl.Err() == nil {
		t.Fatalf("%s: expected Err() to be set", name)
	}
	if !strings.Contains(dl.Err().Message, "row 2") {
		t.Fatalf("%s: expected error to name row 2, got %q", name, dl.Err().Message)
	}
}

func TestInPlaceTransformsRefuseMixedData(t *testing.T) {
	cases := []struct {
		name string
		fn   func(*DataList)
	}{
		{"Normalize", func(dl *DataList) { dl.Normalize() }},
		{"Standardize", func(dl *DataList) { dl.Standardize() }},
		{"ClearOutliers", func(dl *DataList) { dl.ClearOutliers(2) }},
		{"Difference", func(dl *DataList) { dl.Difference() }},
		{"FillNaNWithMean", func(dl *DataList) { dl.FillNaNWithMean() }},
	}
	for _, c := range cases {
		dl := NewDataList(1, "x", 3)
		c.fn(dl)
		assertUntouchedWithErr(t, c.name, dl, []any{1, "x", 3})
	}
}

func TestStandardizeKeepsNaN(t *testing.T) {
	dl := NewDataList(1.0, math.NaN(), 3.0)
	dl.Standardize()
	got := dl.Data()
	if dl.Err() != nil {
		t.Fatalf("unexpected Err: %v", dl.Err())
	}
	if f, ok := got[1].(float64); !ok || !math.IsNaN(f) {
		t.Fatalf("NaN cell should stay NaN, got %v", got)
	}
	want := 1 / math.Sqrt(2)
	if math.Abs(got[0].(float64)+want) > 1e-12 || math.Abs(got[2].(float64)-want) > 1e-12 {
		t.Fatalf("got %v", got)
	}
}

func TestNormalizeKeepsNilAndNaN(t *testing.T) {
	dl := NewDataList(0.0, nil, math.NaN(), 10.0)
	dl.Normalize()
	got := dl.Data()
	if dl.Err() != nil {
		t.Fatalf("unexpected Err: %v", dl.Err())
	}
	if got[1] != nil {
		t.Fatalf("nil should stay nil, got %v", got)
	}
	if f, ok := got[2].(float64); !ok || !math.IsNaN(f) {
		t.Fatalf("NaN should stay NaN, got %v", got)
	}
	if got[0] != 0.0 || got[3] != 1.0 {
		t.Fatalf("got %v", got)
	}
}

func TestClearOutliersKeepsNil(t *testing.T) {
	dl := NewDataList(10, 10, 10, 10, 10, 10, 10, nil, 1000)
	dl.ClearOutliers(2)
	if dl.Err() != nil {
		t.Fatalf("unexpected Err: %v", dl.Err())
	}
	want := []any{10, 10, 10, 10, 10, 10, 10, nil}
	if !reflect.DeepEqual(dl.Data(), want) {
		t.Fatalf("got %v", dl.Data())
	}
}

func TestDifferenceEmitsNaNForMissingOperand(t *testing.T) {
	d := NewDataList(1, nil, 4).Difference()
	if d == nil || d.Len() != 2 {
		t.Fatalf("expected 2 differences, got %v", d)
	}
	for _, v := range d.Data() {
		if f, ok := v.(float64); !ok || !math.IsNaN(f) {
			t.Fatalf("expected NaN, got %v", d.Data())
		}
	}
}

func TestRankRefusesString(t *testing.T) {
	dl := NewDataList(3, "b", 1)
	if got := dl.Rank(); got != nil {
		t.Fatalf("expected nil, got %v", got.Data())
	}
	if dl.Err() == nil {
		t.Fatal("expected Err")
	}
}

func TestRankKeepsNaNPositions(t *testing.T) {
	got := NewDataList(3.0, math.NaN(), 1.0).Rank().Data()
	if got[0] != 2.0 || got[2] != 1.0 {
		t.Fatalf("got %v", got)
	}
	if f, ok := got[1].(float64); !ok || !math.IsNaN(f) {
		t.Fatalf("NaN should rank NaN, got %v", got)
	}
}

func TestSmoothingRefusesString(t *testing.T) {
	dl := NewDataList(1, "2", 3)
	if dl.ExponentialSmoothing(0.5) != nil || dl.Err() == nil {
		t.Fatal("ExponentialSmoothing should fail")
	}
	dl2 := NewDataList(1, "2", 3)
	if dl2.DoubleExponentialSmoothing(0.5, 0.5) != nil || dl2.Err() == nil {
		t.Fatal("DoubleExponentialSmoothing should fail")
	}
}

func TestInterpolationRefusesNil(t *testing.T) {
	cases := map[string]func(*DataList) float64{
		"Linear":          func(dl *DataList) float64 { return dl.LinearInterpolation(0.5) },
		"Quadratic":       func(dl *DataList) float64 { return dl.QuadraticInterpolation(0.5) },
		"Lagrange":        func(dl *DataList) float64 { return dl.LagrangeInterpolation(0.5) },
		"NearestNeighbor": func(dl *DataList) float64 { return dl.NearestNeighborInterpolation(0.5) },
		"Newton":          func(dl *DataList) float64 { return dl.NewtonInterpolation(0.5) },
		"Hermite":         func(dl *DataList) float64 { return dl.HermiteInterpolation(0.5, []float64{1, 1, 1}) },
	}
	for name, fn := range cases {
		dl := NewDataList(1.0, nil, 3.0)
		if got := fn(dl); !math.IsNaN(got) {
			t.Fatalf("%s: expected NaN, got %v", name, got)
		}
		if dl.Err() == nil || !strings.Contains(dl.Err().Message, "row 2") {
			t.Fatalf("%s: expected Err naming row 2, got %v", name, dl.Err())
		}
	}
}
