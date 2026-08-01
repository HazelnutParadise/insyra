package insyra

import (
	"math"
	"reflect"
	"testing"
)

func assertImputeData(t *testing.T, dl *DataList, want []any) {
	t.Helper()
	got := dl.Data()
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d; got %#v", len(got), len(want), got)
	}
	for i := range want {
		if wantF, ok := want[i].(float64); ok && math.IsNaN(wantF) {
			gotF, ok := got[i].(float64)
			if !ok || !math.IsNaN(gotF) {
				t.Fatalf("index %d = %#v, want NaN; got all %#v", i, got[i], got)
			}
			continue
		}
		gotF, gotOK := ToFloat64Safe(got[i])
		wantF, wantOK := ToFloat64Safe(want[i])
		if gotOK && wantOK {
			if math.Abs(gotF-wantF) > 1e-9 {
				t.Fatalf("index %d = %#v, want %#v; got all %#v", i, got[i], want[i], got)
			}
			continue
		}
		if !reflect.DeepEqual(got[i], want[i]) {
			t.Fatalf("index %d = %#v, want %#v; got all %#v", i, got[i], want[i], got)
		}
	}
}

func TestDataListFillForward(t *testing.T) {
	tests := []struct {
		name  string
		input []any
		limit []int
		want  []any
	}{
		{"middle", []any{1.0, math.NaN(), nil, 4.0}, nil, []any{1.0, 1.0, 1.0, 4.0}},
		{"leading", []any{nil, math.NaN(), 2.0, nil}, nil, []any{nil, math.NaN(), 2.0, 2.0}},
		{"limit", []any{1.0, nil, math.NaN(), nil, 5.0}, []int{2}, []any{1.0, 1.0, 1.0, nil, 5.0}},
		{"all missing", []any{nil, math.NaN()}, nil, []any{nil, math.NaN()}},
		{"none missing", []any{1.0, 2.0}, nil, []any{1.0, 2.0}},
		{"single missing", []any{math.NaN()}, nil, []any{math.NaN()}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dl := NewDataList(tt.input...)
			dl.FillForward(tt.limit...)
			assertImputeData(t, dl, tt.want)
		})
	}
}

func TestDataListFillBackward(t *testing.T) {
	tests := []struct {
		name  string
		input []any
		limit []int
		want  []any
	}{
		{"middle", []any{1.0, math.NaN(), nil, 4.0}, nil, []any{1.0, 4.0, 4.0, 4.0}},
		{"trailing", []any{nil, 2.0, nil, math.NaN()}, nil, []any{2.0, 2.0, nil, math.NaN()}},
		{"limit", []any{1.0, nil, math.NaN(), nil, 5.0}, []int{2}, []any{1.0, nil, 5.0, 5.0, 5.0}},
		{"all missing", []any{nil, math.NaN()}, nil, []any{nil, math.NaN()}},
		{"none missing", []any{1.0, 2.0}, nil, []any{1.0, 2.0}},
		{"single missing", []any{nil}, nil, []any{nil}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dl := NewDataList(tt.input...)
			dl.FillBackward(tt.limit...)
			assertImputeData(t, dl, tt.want)
		})
	}
}

func TestDataListFillWithMean(t *testing.T) {
	tests := []struct {
		name  string
		input []any
		want  []any
	}{
		{"middle", []any{1.0, nil, math.NaN(), 4.0}, []any{1.0, 2.5, 2.5, 4.0}},
		{"all missing", []any{nil, math.NaN()}, []any{nil, math.NaN()}},
		{"none missing", []any{1.0, 2.0, 3.0}, []any{1.0, 2.0, 3.0}},
		{"non numeric", []any{"a", nil, math.NaN()}, []any{"a", nil, math.NaN()}},
		{"handles nil unlike FillNaNWithMean", []any{nil, 2.0, 4.0}, []any{3.0, 2.0, 4.0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dl := NewDataList(tt.input...)
			dl.FillWithMean()
			assertImputeData(t, dl, tt.want)
		})
	}
}

func TestDataListFillWithMedian(t *testing.T) {
	tests := []struct {
		name  string
		input []any
		want  []any
	}{
		{"middle", []any{1.0, nil, math.NaN(), 3.0, 100.0}, []any{1.0, 3.0, 3.0, 3.0, 100.0}},
		{"all missing", []any{nil, math.NaN()}, []any{nil, math.NaN()}},
		{"none missing", []any{1.0, 2.0, 3.0}, []any{1.0, 2.0, 3.0}},
		{"non numeric", []any{"a", nil, math.NaN()}, []any{"a", nil, math.NaN()}},
		{"single observed", []any{nil, 5.0}, []any{5.0, 5.0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dl := NewDataList(tt.input...)
			dl.FillWithMedian()
			assertImputeData(t, dl, tt.want)
		})
	}
}

func TestDataListFillWithMode(t *testing.T) {
	tests := []struct {
		name  string
		input []any
		want  []any
	}{
		{"strings", []any{"a", nil, "b", "a", math.NaN()}, []any{"a", "a", "b", "a", "a"}},
		{"multiple modes first wins", []any{"b", "a", "a", "b", nil}, []any{"b", "a", "a", "b", "b"}},
		{"all missing", []any{nil, math.NaN()}, []any{nil, math.NaN()}},
		{"none missing", []any{"x", "y"}, []any{"x", "y"}},
		{"single observed", []any{nil, "x"}, []any{"x", "x"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dl := NewDataList(tt.input...)
			dl.FillWithMode()
			assertImputeData(t, dl, tt.want)
		})
	}
}

func TestDataListFillByInterpolation(t *testing.T) {
	tests := []struct {
		name        string
		input       []any
		extrapolate []bool
		want        []any
	}{
		{"middle", []any{1.0, math.NaN(), nil, 4.0}, nil, []any{1.0, 2.0, 3.0, 4.0}},
		{"edges", []any{nil, 1.0, nil, 3.0, math.NaN()}, nil, []any{nil, 1.0, 2.0, 3.0, math.NaN()}},
		{"extrapolate", []any{nil, 1.0, nil, 3.0, math.NaN()}, []bool{true}, []any{0.0, 1.0, 2.0, 3.0, 4.0}},
		{"non numeric", []any{"a", nil, "b"}, nil, []any{"a", nil, "b"}},
		{"all missing", []any{nil, math.NaN()}, nil, []any{nil, math.NaN()}},
		{"none missing", []any{1.0, 2.0}, nil, []any{1.0, 2.0}},
		{"single missing", []any{nil}, nil, []any{nil}},
		{"single observed extrapolate", []any{nil, 5.0, math.NaN()}, []bool{true}, []any{5.0, 5.0, 5.0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dl := NewDataList(tt.input...)
			dl.FillByInterpolation(tt.extrapolate...)
			assertImputeData(t, dl, tt.want)
		})
	}
}
