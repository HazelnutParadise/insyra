package ta

import (
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// The 33 closes of the StockCharts RSI worked example. Wilder's RSI(14) on
// them is 70.53, 66.32, 66.55, 69.41, 66.36, 57.97, ... from row index 14; the
// simple (Cutler) values were computed separately in Python from the
// definition.
var rsiPrices = []any{44.3389, 44.0902, 44.1497, 43.6124, 44.3278, 44.8264, 45.0955, 45.4245, 45.8433, 46.0826, 45.8931, 46.0328, 45.6140, 46.2820, 46.2820, 46.0028, 46.0328, 46.4116, 46.2222, 45.6439, 46.2122, 46.2521, 45.7137, 46.4515, 45.7835, 45.3548, 44.0288, 44.1783, 44.2181, 44.5672, 43.4205, 42.6628, 43.1314}

func roundedFrom(t *testing.T, dl *insyra.DataList, from, count int, places float64) []float64 {
	t.Helper()
	all := floats(t, dl)
	out := make([]float64, count)
	for i := range out {
		out[i] = math.Round(all[from+i]*places) / places
	}
	return out
}

func TestRSIWilderMatchesTheWorkedExample(t *testing.T) {
	got, err := RSI(insyra.NewDataList(rsiPrices...), 14)
	if err != nil {
		t.Fatal(err)
	}
	all := floats(t, got)
	for i := 0; i < 14; i++ {
		if !math.IsNaN(all[i]) {
			t.Errorf("row index %d should be warm-up NaN, got %v", i, all[i])
		}
	}
	want := []float64{70.53, 66.32, 66.55, 69.41, 66.36, 57.97, 62.93, 63.26, 56.06, 62.38, 54.71, 50.42, 39.99, 41.46, 41.87, 45.46, 37.3, 33.08, 37.77}
	g := roundedFrom(t, got, 14, len(want), 100)
	for i := range want {
		if g[i] != want[i] {
			t.Errorf("row index %d: got %v, want %v", 14+i, g[i], want[i])
		}
	}
}

func TestRSISimple(t *testing.T) {
	got, err := RSI(insyra.NewDataList(rsiPrices...), 14, RSIOptions{Method: RSISimple})
	if err != nil {
		t.Fatal(err)
	}
	want := []float64{70.5328, 70.0811, 69.8946, 80.5977, 73.3981, 59.9021}
	g := roundedFrom(t, got, 14, len(want), 10000)
	for i := range want {
		if g[i] != want[i] {
			t.Errorf("row index %d: got %v, want %v", 14+i, g[i], want[i])
		}
	}
}

func TestRSIEdgeValues(t *testing.T) {
	up, _ := RSI(insyra.NewDataList(1.0, 2.0, 3.0, 4.0), 3)
	if v := floats(t, up)[3]; v != 100 {
		t.Errorf("only gains: got %v, want 100", v)
	}
	down, _ := RSI(insyra.NewDataList(4.0, 3.0, 2.0, 1.0), 3)
	if v := floats(t, down)[3]; v != 0 {
		t.Errorf("only losses: got %v, want 0", v)
	}
	flat, _ := RSI(insyra.NewDataList(2.0, 2.0, 2.0, 2.0), 3)
	if v := floats(t, flat)[3]; !math.IsNaN(v) {
		t.Errorf("no change: got %v, want NaN", v)
	}
	short, _ := RSI(insyra.NewDataList(1.0, 2.0), 3)
	for _, v := range floats(t, short) {
		if !math.IsNaN(v) {
			t.Errorf("fewer than n+1 prices should be all NaN, got %v", v)
		}
	}
}

func TestRSIMissingPrices(t *testing.T) {
	gap := insyra.NewDataList(1.0, 2.0, nil, 4.0, 3.0, 5.0)
	if _, err := RSI(gap, 2); err == nil {
		t.Error("a gap is an error by default")
	}
	// Flat: the gap carries 2.0 over, so the series is 1, 2, 2, 4, 3, 5.
	flat, err := RSI(gap, 2, RSIOptions{Missing: RSIMissingFlat})
	if err != nil {
		t.Fatal(err)
	}
	ref, _ := RSI(insyra.NewDataList(1.0, 2.0, 2.0, 4.0, 3.0, 5.0), 2, RSIOptions{Method: RSIWilder})
	f, r := floats(t, flat), floats(t, ref)
	for i := range f {
		if i == 2 {
			if !math.IsNaN(f[i]) {
				t.Errorf("the missing day itself should be NaN, got %v", f[i])
			}
			continue
		}
		if !(f[i] == r[i] || (math.IsNaN(f[i]) && math.IsNaN(r[i]))) {
			t.Errorf("row index %d: got %v, want %v", i, f[i], r[i])
		}
	}
	// Break: after the gap, 4, 3, 5 is a new series that needs 2 changes.
	brk, err := RSI(gap, 2, RSIOptions{Missing: RSIMissingBreak})
	if err != nil {
		t.Fatal(err)
	}
	b := floats(t, brk)
	if !math.IsNaN(b[2]) || !math.IsNaN(b[3]) || !math.IsNaN(b[4]) || math.IsNaN(b[5]) {
		t.Errorf("break: got %v", b)
	}
}

func TestRSIRefusesBadInput(t *testing.T) {
	good := insyra.NewDataList(1.0, 2.0, 3.0)
	var nilList *insyra.DataList
	cases := map[string]func() error{
		"n":       func() error { _, e := RSI(good, 0); return e },
		"nil":     func() error { _, e := RSI(nilList, 2); return e },
		"inf":     func() error { _, e := RSI(insyra.NewDataList(1.0, math.Inf(1)), 1); return e },
		"text":    func() error { _, e := RSI(insyra.NewDataList(1.0, "x"), 1); return e },
		"method":  func() error { _, e := RSI(good, 1, RSIOptions{Method: RSIMethod(9)}); return e },
		"missing": func() error { _, e := RSI(good, 1, RSIOptions{Missing: RSIMissing(9)}); return e },
		"two":     func() error { _, e := RSI(good, 1, RSIOptions{}, RSIOptions{}); return e },
	}
	for name, f := range cases {
		if f() == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}

// A segment start restarts RSI: the rows after it need n changes of their own,
// and nothing leaks across the break.
func TestRSIRestartsAtASegmentStart(t *testing.T) {
	p := []float64{10, 11, 12, 11, 13, 14, 20, 21, 20, 22, 23, 21}
	whole, err := RSI(prices(p...), 2)
	if err != nil {
		t.Fatal(err)
	}
	split, err := RSI(prices(p...), 2, RSIOptions{SegmentStarts: []int{6}})
	if err != nil {
		t.Fatal(err)
	}
	w, s := floats(t, whole), floats(t, split)
	for i := 0; i < 6; i++ {
		if !(w[i] == s[i] || (math.IsNaN(w[i]) && math.IsNaN(s[i]))) {
			t.Fatalf("row %d before the break changed: %v vs %v", i, w[i], s[i])
		}
	}
	// Rows 6 and 7 have fewer than 2 changes in the new segment.
	if !math.IsNaN(s[6]) || !math.IsNaN(s[7]) || math.IsNaN(s[8]) {
		t.Fatalf("the new segment starts again: %v", s[6:])
	}
	// The second segment on its own gives the same numbers.
	alone, _ := RSI(prices(p[6:]...), 2)
	a := floats(t, alone)
	for i := range a {
		if !(a[i] == s[6+i] || (math.IsNaN(a[i]) && math.IsNaN(s[6+i]))) {
			t.Fatalf("segment alone %v, in context %v", a, s[6:])
		}
	}
	if _, err := RSI(prices(p...), 2, RSIOptions{SegmentStarts: []int{12}}); err == nil {
		t.Error("a segment start outside the series should be an error")
	}
}
