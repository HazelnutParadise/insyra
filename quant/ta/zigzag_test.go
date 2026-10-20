package ta

import (
	"math"
	"math/rand"
	"reflect"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

func prices(v ...float64) *insyra.DataList {
	a := make([]any, len(v))
	for i := range v {
		a[i] = v[i]
	}
	return insyra.NewDataList(a...)
}

// The worked example of issue 442: L@0 confirmed@1, H@2 confirmed@3, L@4
// confirmed@6, H@7 confirmed@8.
func TestZigZagWorkedExample(t *testing.T) {
	got, err := ZigZag(prices(100, 111, 120, 105, 95, 100, 115, 130, 116), 0.10)
	if err != nil {
		t.Fatal(err)
	}
	want := []Swing{
		{SwingLow, 0, 100, 1, 0},
		{SwingHigh, 2, 120, 3, 0},
		{SwingLow, 4, 95, 6, 0},
		{SwingHigh, 7, 130, 8, 0},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
}

// A swing found on a series cut short is exactly a swing of the full series
// that was confirmed before the cut.
func TestZigZagIsCausal(t *testing.T) {
	rng := rand.New(rand.NewSource(11))
	series := make([]float64, 400)
	series[0] = 100
	for i := 1; i < len(series); i++ {
		series[i] = series[i-1] * math.Exp(0.03*rng.NormFloat64())
	}
	full, err := ZigZag(prices(series...), 0.08)
	if err != nil || len(full) < 5 {
		t.Fatalf("expected several swings, got %d, %v", len(full), err)
	}
	for cut := 1; cut <= len(series); cut += 7 {
		part, err := ZigZag(prices(series[:cut]...), 0.08)
		if err != nil {
			t.Fatal(err)
		}
		var want []Swing
		for _, pv := range full {
			if pv.ConfirmedAt < cut {
				want = append(want, pv)
			}
		}
		if !reflect.DeepEqual(part, want) {
			t.Fatalf("cut at %d: got %+v, want %+v", cut, part, want)
		}
	}
}

func TestZigZagStartsWithAFallAndAlternates(t *testing.T) {
	got, err := ZigZag(prices(100, 95, 85, 100, 120, 100), 0.10)
	if err != nil {
		t.Fatal(err)
	}
	want := []Swing{{SwingHigh, 0, 100, 2, 0}, {SwingLow, 2, 85, 3, 0}, {SwingHigh, 4, 120, 5, 0}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
	flat, _ := ZigZag(prices(100, 101, 100, 102), 0.10)
	if len(flat) != 0 {
		t.Errorf("small moves are not swings: %+v", flat)
	}
}

func TestZigZagRefusesBadInput(t *testing.T) {
	var nilList *insyra.DataList
	cases := map[string]func() error{
		"threshold 0":   func() error { _, e := ZigZag(prices(1, 2), 0); return e },
		"threshold 1":   func() error { _, e := ZigZag(prices(1, 2), 1); return e },
		"threshold NaN": func() error { _, e := ZigZag(prices(1, 2), math.NaN()); return e },
		"zero price":    func() error { _, e := ZigZag(prices(1, 0), 0.1); return e },
		"nil":           func() error { _, e := ZigZag(nilList, 0.1); return e },
		"NaN":           func() error { _, e := ZigZag(prices(1, math.NaN()), 0.1); return e },
	}
	for name, f := range cases {
		if f() == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}

// sameRegimes compares regimes, allowing the last bits of Change to differ.
func sameRegimes(a, b []Regime) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		x, y := a[i], b[i]
		if math.Abs(x.Change-y.Change) > 1e-12 {
			return false
		}
		x.Change, y.Change = 0, 0
		if x != y {
			return false
		}
	}
	return true
}

func TestRegimes(t *testing.T) {
	swings, _ := ZigZag(prices(100, 111, 120, 105, 95, 100, 115, 130, 116), 0.10)
	got, err := Regimes(swings)
	if err != nil {
		t.Fatal(err)
	}
	want := []Regime{
		{RegimeBull, 0, 2, 2, 0.20, 3},
		{RegimeBear, 2, 4, 2, 95.0/120 - 1, 6},
		{RegimeBull, 4, 7, 3, 130.0/95 - 1, 8},
	}
	if !sameRegimes(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
}

func TestRegimesMinimums(t *testing.T) {
	swings := []Swing{
		{SwingLow, 0, 100, 3, 0}, {SwingHigh, 10, 150, 12, 0}, {SwingLow, 12, 140, 14, 0},
		{SwingHigh, 25, 190, 27, 0}, {SwingLow, 40, 120, 42, 0},
	}
	// The bear stretch 10 -> 12 is 2 rows long; with MinPhase 3 it goes, and the
	// two bull stretches around it join into one from 0 to 25.
	phases, err := Regimes(swings, RegimeOptions{MinPhase: 3})
	if err != nil {
		t.Fatal(err)
	}
	want := []Regime{{RegimeBull, 0, 25, 25, 0.9, 27}, {RegimeBear, 25, 40, 15, 120.0/190 - 1, 42}}
	if !sameRegimes(phases, want) {
		t.Errorf("MinPhase: got %+v\nwant %+v", phases, want)
	}
	// Lows at 0 and 12 make a cycle of 12 rows; MinCycle 13 removes the high at 10 and the low at 12.
	cycles, err := Regimes(swings, RegimeOptions{MinCycle: 13})
	if err != nil {
		t.Fatal(err)
	}
	if !sameRegimes(cycles, want) {
		t.Errorf("MinCycle: got %+v\nwant %+v", cycles, want)
	}
	if _, err := Regimes([]Swing{{SwingLow, 0, 1, 1, 0}, {SwingLow, 2, 1, 3, 0}}); err == nil {
		t.Error("swings that do not alternate should be an error")
	}
	if _, err := Regimes(swings, RegimeOptions{MinPhase: -1}); err == nil {
		t.Error("a negative minimum should be an error")
	}
}
