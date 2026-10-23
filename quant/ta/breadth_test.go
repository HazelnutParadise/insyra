package ta

import (
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

func sameSeries(t *testing.T, got *insyra.DataList, want []float64) {
	t.Helper()
	g := floats(t, got)
	if len(g) != len(want) {
		t.Fatalf("got %v, want %v", g, want)
	}
	for i := range want {
		if math.IsNaN(want[i]) != math.IsNaN(g[i]) || (!math.IsNaN(want[i]) && math.Abs(g[i]-want[i]) > 1e-12) {
			t.Fatalf("row index %d: got %v, want %v (all: %v)", i, g[i], want[i], g)
		}
	}
}

var (
	breadthAdv = []any{10.0, 20.0, 30.0, 0.0, 5.0}
	breadthDec = []any{5.0, 20.0, 10.0, 0.0, 15.0}
	nan        = math.NaN()
)

func TestBreadthIndicators(t *testing.T) {
	adv, dec := insyra.NewDataList(breadthAdv...), insyra.NewDataList(breadthDec...)
	adr, err := AdvanceDeclineRatio(adv, dec, 2)
	if err != nil {
		t.Fatal(err)
	}
	// windows: (10+20)/(5+20), (20+30)/(20+10), (30+0)/(10+0), (0+5)/(0+15)
	sameSeries(t, adr, []float64{nan, 30.0 / 25, 50.0 / 30, 3, 5.0 / 15})
	obos, _ := OverboughtOversold(adv, dec, 2)
	sameSeries(t, obos, []float64{nan, 5, 20, 20, -10})
	net, _ := NetAdvanceRatio(adv, dec, 2)
	sameSeries(t, net, []float64{nan, 5.0 / 55, 20.0 / 80, 20.0 / 40, -10.0 / 20})
	adl, _ := AdvanceDeclineLine(adv, dec, 100)
	sameSeries(t, adl, []float64{105, 105, 125, 125, 115})
}

func TestBreadthZeroDenominatorsAreNaN(t *testing.T) {
	adv, dec := insyra.NewDataList(0.0, 3.0), insyra.NewDataList(0.0, 0.0)
	adr, _ := AdvanceDeclineRatio(adv, dec, 1)
	sameSeries(t, adr, []float64{nan, nan})
	net, _ := NetAdvanceRatio(adv, dec, 1)
	sameSeries(t, net, []float64{nan, 1})
}

func TestBreadthMissingDaysAreSkippedNotZeroed(t *testing.T) {
	adv := insyra.NewDataList(10.0, nil, 20.0, 30.0)
	dec := insyra.NewDataList(5.0, 7.0, math.NaN(), 10.0)
	// valid days: 0 and 3 only
	obos, err := OverboughtOversold(adv, dec, 2)
	if err != nil {
		t.Fatal(err)
	}
	sameSeries(t, obos, []float64{nan, nan, nan, 25})
	adl, _ := AdvanceDeclineLine(adv, dec, 0)
	sameSeries(t, adl, []float64{5, nan, nan, 25})
}

func TestBreadthRefusesBadInput(t *testing.T) {
	good := insyra.NewDataList(1.0, 2.0)
	var nilList *insyra.DataList
	cases := map[string]func() error{
		"negative":   func() error { _, e := OverboughtOversold(insyra.NewDataList(-1.0, 2.0), good, 1); return e },
		"inf":        func() error { _, e := OverboughtOversold(insyra.NewDataList(math.Inf(1), 2.0), good, 1); return e },
		"text":       func() error { _, e := OverboughtOversold(insyra.NewDataList("x", 2.0), good, 1); return e },
		"mismatch":   func() error { _, e := OverboughtOversold(good, insyra.NewDataList(1.0), 1); return e },
		"nil":        func() error { _, e := OverboughtOversold(nilList, good, 1); return e },
		"window":     func() error { _, e := AdvanceDeclineRatio(good, good, 0); return e },
		"start":      func() error { _, e := AdvanceDeclineLine(good, good, math.NaN()); return e },
		"net window": func() error { _, e := NetAdvanceRatio(good, good, -1); return e },
	}
	for name, f := range cases {
		if f() == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}
