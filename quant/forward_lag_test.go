package quant

import (
	"fmt"
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

// lagStudy is one stock over eight months of three rows each, with an event on
// the first row of every month. With Delay 1 and Horizon 1 each event earns
// the return from its second to its third row; a flat benchmark leaves that
// return as the excess, so the monthly series is returns.
func lagStudy() (prices *insyra.DataTable, events *insyra.DataTable, dates *insyra.DataList, flat *insyra.DataList, returns []float64) {
	rets := []float64{0.02, 0.05, -0.01, 0.04, 0.03, -0.02, 0.06, 0.01}
	px := make([]any, 0, 24)
	flatLevels := make([]any, 0, 24)
	days := make([]any, 0, 24)
	ev := make([]any, 0, 24)
	level := 100.0
	for m, r := range rets {
		px = append(px, level, level, level*(1+r))
		level *= 1 + r
		flatLevels = append(flatLevels, 100.0, 100.0, 100.0)
		for d := 1; d <= 3; d++ {
			days = append(days, fmt.Sprintf("2026-%02d-%02d", m+1, d))
		}
		ev = append(ev, true, false, false)
	}
	return insyra.NewDataTable(insyra.NewDataList(px...).SetName("A")),
		insyra.NewDataTable(insyra.NewDataList(ev...).SetName("A")),
		insyra.NewDataList(days...), insyra.NewDataList(flatLevels...), rets
}

func lagTStat(t *testing.T, o ForwardOptions) ForwardSummary {
	t.Helper()
	prices, events, dates, flat, _ := lagStudy()
	o.Horizons = []int{1}
	o.Benchmark = BenchmarkLevels
	o.BenchmarkLevels = flat
	res, err := EventStudy(prices, events, dates, EventStudyOptions{ForwardOptions: o})
	if err != nil {
		t.Fatal(err)
	}
	return res[0]
}

func TestForwardNeweyWestLagRules(t *testing.T) {
	_, _, _, _, rets := lagStudy()
	series := make([]any, len(rets))
	for i, r := range rets {
		series[i] = r
	}
	want := func(lag int) *stats.NeweyWestResult {
		res, err := stats.NeweyWestMeanTest(insyra.NewDataList(series...), lag)
		if err != nil {
			t.Fatal(err)
		}
		return res
	}

	auto := lagTStat(t, ForwardOptions{})
	if ref := want(stats.NeweyWestLag(8)); auto.Lag != ref.Lag || math.Abs(auto.TStat-ref.Statistic) > 1e-9 {
		t.Errorf("automatic: lag %d t %v, want lag %d t %v", auto.Lag, auto.TStat, ref.Lag, ref.Statistic)
	}
	given := lagTStat(t, ForwardOptions{NeweyWestLag: 2})
	if ref := want(2); given.Lag != 2 || math.Abs(given.TStat-ref.Statistic) > 1e-9 {
		t.Errorf("lag 2: lag %d t %v, want t %v", given.Lag, given.TStat, ref.Statistic)
	}
	// NoNeweyWest is the t-test with lag 0 and a divisor of T: what a negative
	// lag meant before it became an error.
	off := lagTStat(t, ForwardOptions{NoNeweyWest: true})
	ref := want(0)
	if off.Lag != 0 || math.Abs(off.TStat-ref.Statistic) > 1e-9 || math.Abs(off.PValue-ref.PValue) > 1e-12 {
		t.Errorf("NoNeweyWest: lag %d t %v p %v, want t %v p %v", off.Lag, off.TStat, off.PValue, ref.Statistic, ref.PValue)
	}
	if off.TStat == auto.TStat {
		t.Errorf("switching the correction off should change t, both %v", off.TStat)
	}
}

func TestForwardNeweyWestLagErrors(t *testing.T) {
	prices, events, dates, flat, _ := lagStudy()
	cases := map[string]ForwardOptions{
		"negative lag":            {NeweyWestLag: -1},
		"NoNeweyWest with a lag":  {NoNeweyWest: true, NeweyWestLag: 2},
		"NoNeweyWest with -1 lag": {NoNeweyWest: true, NeweyWestLag: -1},
	}
	for name, o := range cases {
		o.Horizons = []int{1}
		o.Benchmark = BenchmarkLevels
		o.BenchmarkLevels = flat
		if _, err := EventStudy(prices, events, dates, EventStudyOptions{ForwardOptions: o}); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}
