package insyra

import (
	"math"
	"testing"
)

// The expected values were computed with pandas:
// pd.Series([1, 2, 3, 4, 10]).quantile([0, .25, .5, .75, 1]) and the
// pd.Series([1, 2, 3, 4]).quantile(.25) example of pandas' own documentation.
func TestQuantileMatchesPandas(t *testing.T) {
	dl := NewDataList(1, 2, 3, 4, 10)
	for _, tc := range []struct{ p, want float64 }{
		{0, 1}, {0.25, 2}, {0.5, 3}, {0.75, 4}, {1, 10},
	} {
		if got := dl.Quantile(tc.p); got != tc.want {
			t.Errorf("Quantile(%v) = %v, want %v", tc.p, got, tc.want)
		}
	}
	if got := NewDataList(1, 2, 3, 4).Quantile(0.25); got != 1.75 {
		t.Errorf("Quantile(0.25) of [1 2 3 4] = %v, want 1.75", got)
	}
	if err := dl.Err(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestQuantileAgreesWithPercentileAndQuartile(t *testing.T) {
	dl := NewDataList(3.5, -1, 8, 2.25, 9, 4, 0, 7.5)
	for _, p := range []float64{0, 0.1, 0.25, 0.5, 0.75, 0.9, 1} {
		if q, pc := dl.Quantile(p), dl.Percentile(p*100); q != pc {
			t.Errorf("Quantile(%v) = %v, Percentile(%v) = %v", p, q, p*100, pc)
		}
	}
	for q, p := range map[int]float64{1: 0.25, 2: 0.5, 3: 0.75} {
		if got, want := dl.Quartile(q), dl.Quantile(p); got != want {
			t.Errorf("Quartile(%d) = %v, Quantile(%v) = %v", q, got, p, want)
		}
	}
}

func TestQuantileSkipsNilAndNaN(t *testing.T) {
	for _, dl := range []*DataList{
		NewDataList(1, 2, 3, math.NaN()),
		NewDataList(math.NaN(), 1, nil, 2, 3),
	} {
		if got := dl.Quantile(0.25); got != 1.5 {
			t.Errorf("Quantile(0.25) = %v, want 1.5", got)
		}
		if got := dl.Quantile(0.75); got != 2.5 {
			t.Errorf("Quantile(0.75) = %v, want 2.5", got)
		}
		if err := dl.Err(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	}
}

func TestQuantileRefusesWhatItCannotAnswer(t *testing.T) {
	bad := []struct {
		name string
		dl   *DataList
		p    float64
	}{
		{"p below 0", NewDataList(1, 2, 3), -0.1},
		{"p above 1", NewDataList(1, 2, 3), 1.1},
		{"p is NaN", NewDataList(1, 2, 3), math.NaN()},
		{"a text cell", NewDataList(1, "x", 3), 0.5},
		{"no numeric values", NewDataList(nil, math.NaN()), 0.5},
		{"empty", NewDataList(), 0.5},
	}
	for _, tc := range bad {
		if got := tc.dl.Quantile(tc.p); !math.IsNaN(got) {
			t.Errorf("%s: Quantile(%v) = %v, want NaN", tc.name, tc.p, got)
		}
		if tc.dl.Err() == nil {
			t.Errorf("%s: no warning was recorded", tc.name)
		}
	}
}
