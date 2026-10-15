package insyra

import (
	"math"
	"reflect"
	"testing"
)

// #221 (D-13): Mode returns every value that reaches the highest count, in
// ascending order, the way pandas' Series.mode does. Before, a list whose
// values all had the same count — [1 2 3], [1 1 2 2], even [7] — returned
// nil, and the order of several modes followed map iteration.
// The expected values were checked with pandas 3.0.5.
func TestModeReturnsEveryModeLikePandas(t *testing.T) {
	cases := []struct {
		data []any
		want []float64
	}{
		{[]any{1, 2, 3}, []float64{1, 2, 3}},
		{[]any{1, 1, 2, 2}, []float64{1, 2}},
		{[]any{7}, []float64{7}},
		{[]any{3, 3, 1, 1, 2}, []float64{1, 3}},
		{[]any{1, 2, 2, 3, 3, 3}, []float64{3}},
		{[]any{math.NaN(), math.NaN(), 1}, []float64{1}},
		{[]any{nil, nil, 5, 4, 5}, []float64{5}},
		{[]any{2, 2.0, int64(1)}, []float64{2}},
		{[]any{-1.5, 10, -1.5, 10, 0}, []float64{-1.5, 10}},
	}
	for _, tc := range cases {
		dl := NewDataList(tc.data...)
		if got := dl.Mode(); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("Mode(%v) = %v, want %v", tc.data, got, tc.want)
		}
		if err := dl.Err(); err != nil {
			t.Errorf("Mode(%v): unexpected error %v", tc.data, err)
		}
	}
}

func TestModeOrderIsStable(t *testing.T) {
	values := make([]any, 0, 40)
	for i := 20; i > 0; i-- {
		values = append(values, i, i)
	}
	dl := NewDataList(values...)
	want := dl.Mode()
	for i := 0; i < 50; i++ {
		if got := dl.Mode(); !reflect.DeepEqual(got, want) {
			t.Fatalf("Mode changed between calls: %v then %v", want, got)
		}
	}
	for i := 1; i < len(want); i++ {
		if want[i-1] >= want[i] {
			t.Fatalf("Mode is not ascending: %v", want)
		}
	}
}

func TestModeOfNothingIsEmpty(t *testing.T) {
	for _, dl := range []*DataList{NewDataList(), NewDataList(nil, math.NaN()), NewDataList("a", "b")} {
		if got := dl.Mode(); len(got) != 0 {
			t.Errorf("Mode(%v) = %v, want no values", dl.Data(), got)
		}
		if err := dl.Err(); err != nil {
			t.Errorf("Mode(%v): unexpected error %v", dl.Data(), err)
		}
	}
}
