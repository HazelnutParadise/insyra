package quant

import (
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

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
