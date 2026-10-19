package quant

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
	want := []Pivot{
		{PivotLow, 0, 100, 1, 0},
		{PivotHigh, 2, 120, 3, 0},
		{PivotLow, 4, 95, 6, 0},
		{PivotHigh, 7, 130, 8, 0},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
}

// A pivot found on a series cut short is exactly a pivot of the full series
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
		t.Fatalf("expected several pivots, got %d, %v", len(full), err)
	}
	for cut := 1; cut <= len(series); cut += 7 {
		part, err := ZigZag(prices(series[:cut]...), 0.08)
		if err != nil {
			t.Fatal(err)
		}
		var want []Pivot
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
	want := []Pivot{{PivotHigh, 0, 100, 2, 0}, {PivotLow, 2, 85, 3, 0}, {PivotHigh, 4, 120, 5, 0}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
	flat, _ := ZigZag(prices(100, 101, 100, 102), 0.10)
	if len(flat) != 0 {
		t.Errorf("small moves are not pivots: %+v", flat)
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
	pivots, _ := ZigZag(prices(100, 111, 120, 105, 95, 100, 115, 130, 116), 0.10)
	got, err := Regimes(pivots)
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
	pivots := []Pivot{
		{PivotLow, 0, 100, 3, 0}, {PivotHigh, 10, 150, 12, 0}, {PivotLow, 12, 140, 14, 0},
		{PivotHigh, 25, 190, 27, 0}, {PivotLow, 40, 120, 42, 0},
	}
	// The bear stretch 10 -> 12 is 2 rows long; with MinPhase 3 it goes, and the
	// two bull stretches around it join into one from 0 to 25.
	phases, err := Regimes(pivots, RegimeOptions{MinPhase: 3})
	if err != nil {
		t.Fatal(err)
	}
	want := []Regime{{RegimeBull, 0, 25, 25, 0.9, 27}, {RegimeBear, 25, 40, 15, 120.0/190 - 1, 42}}
	if !sameRegimes(phases, want) {
		t.Errorf("MinPhase: got %+v\nwant %+v", phases, want)
	}
	// Lows at 0 and 12 make a cycle of 12 rows; MinCycle 13 removes the high at 10 and the low at 12.
	cycles, err := Regimes(pivots, RegimeOptions{MinCycle: 13})
	if err != nil {
		t.Fatal(err)
	}
	if !sameRegimes(cycles, want) {
		t.Errorf("MinCycle: got %+v\nwant %+v", cycles, want)
	}
	if _, err := Regimes([]Pivot{{PivotLow, 0, 1, 1, 0}, {PivotLow, 2, 1, 3, 0}}); err == nil {
		t.Error("pivots that do not alternate should be an error")
	}
	if _, err := Regimes(pivots, RegimeOptions{MinPhase: -1}); err == nil {
		t.Error("a negative minimum should be an error")
	}
}

func nearlyAll(t *testing.T, name string, got *insyra.DataList, want []float64) {
	t.Helper()
	g := floats(t, got)
	if len(g) != len(want) {
		t.Fatalf("%s: got %v, want %v", name, g, want)
	}
	for i := range want {
		if math.Abs(g[i]-want[i]) > 1e-12 {
			t.Fatalf("%s: row index %d: got %v, want %v (all %v)", name, i, g[i], want[i], g)
		}
	}
}

// A signal of 1,1,0,0,1 is in force one row later: 0,1,1,0,0. Buying costs
// 0.002 on row 1, selling 0.001 on row 3, and the cash earns 0.001 whenever it
// is held.
func TestBacktestSignalDelaysTheSignalAndChargesAsymmetricCosts(t *testing.T) {
	res, err := BacktestSignal(
		prices(1, 1, 0, 0, 1), prices(0.01, 0.02, -0.03, 0.01, 0.02),
		BacktestOptions{CashReturns: prices(0.001, 0.001, 0.001, 0.001, 0.001), BuyCost: 0.002, SellCost: 0.001})
	if err != nil {
		t.Fatal(err)
	}
	nearlyAll(t, "positions", res.Positions, []float64{0, 1, 1, 0, 0})
	nearlyAll(t, "net", res.NetReturns, []float64{0.001, 0.018, -0.03, 0.0, 0.001})
	nearlyAll(t, "turnover", res.Turnover, []float64{0, 1, 0, 1, 0})
	if res.Trades != 2 {
		t.Errorf("trades: %d", res.Trades)
	}
}

func TestBacktestSignalFractionalPositionsAndInitialPosition(t *testing.T) {
	res, err := BacktestSignal(prices(0.5, 0.5, 0), prices(0.10, 0.10, 0.10),
		BacktestOptions{BuyCost: 0.01, SellCost: 0.02, InitialPosition: 1})
	if err != nil {
		t.Fatal(err)
	}
	// Held: 1, 0.5, 0.5. Row 0 stays at 1: 0.10. Row 1 sells half: 0.05 - 0.5*0.02. Row 2 holds: 0.05.
	nearlyAll(t, "net", res.NetReturns, []float64{0.10, 0.04, 0.05})
	if res.Trades != 1 {
		t.Errorf("trades: %d", res.Trades)
	}
}

func TestBacktestSignalWithoutCashOrCostsIsPositionTimesReturn(t *testing.T) {
	res, err := BacktestSignal(prices(1, 0, 1), prices(0.1, 0.2, 0.3))
	if err != nil {
		t.Fatal(err)
	}
	nearlyAll(t, "net", res.NetReturns, []float64{0, 0.2, 0})
}

func TestBacktestSignalRefusesBadInput(t *testing.T) {
	var nilList *insyra.DataList
	cases := map[string]func() error{
		"signal range":  func() error { _, e := BacktestSignal(prices(1.5, 0), prices(0, 0)); return e },
		"length":        func() error { _, e := BacktestSignal(prices(1, 0), prices(0)); return e },
		"nil":           func() error { _, e := BacktestSignal(nilList, prices(0)); return e },
		"NaN return":    func() error { _, e := BacktestSignal(prices(1), prices(math.NaN())); return e },
		"cost":          func() error { _, e := BacktestSignal(prices(1), prices(0), BacktestOptions{BuyCost: 1}); return e },
		"negative cost": func() error { _, e := BacktestSignal(prices(1), prices(0), BacktestOptions{SellCost: -0.1}); return e },
		"initial": func() error {
			_, e := BacktestSignal(prices(1), prices(0), BacktestOptions{InitialPosition: 2})
			return e
		},
		"cash length": func() error {
			_, e := BacktestSignal(prices(1, 1), prices(0, 0), BacktestOptions{CashReturns: prices(0)})
			return e
		},
		"two options": func() error {
			_, e := BacktestSignal(prices(1), prices(0), BacktestOptions{}, BacktestOptions{})
			return e
		},
	}
	for name, f := range cases {
		if f() == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}
