package quant

import (
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/quant/ta"
)

// Two stocks over six dates in two months. Stock A rises 10% on four of the
// five steps and is flat on the third; B never moves.
func studyData() (prices *insyra.DataTable, dates *insyra.DataList) {
	a := insyra.NewDataList(100.0, 100.0, 110.0, 121.0, 121.0, 133.1).SetName("A")
	b := insyra.NewDataList(100.0, 100.0, 100.0, 100.0, 100.0, 100.0).SetName("B")
	d := insyra.NewDataList("2026-01-29", "2026-01-30", "2026-02-02", "2026-02-03", "2026-02-04", "2026-02-05")
	return insyra.NewDataTable(a, b), d
}

func flagTable(a, b []bool) *insyra.DataTable {
	conv := func(v []bool) *insyra.DataList {
		out := make([]any, len(v))
		for i := range v {
			out[i] = v[i]
		}
		return insyra.NewDataList(out...)
	}
	return insyra.NewDataTable(conv(a).SetName("A"), conv(b).SetName("B"))
}

func near(a, b float64) bool { return math.Abs(a-b) < 1e-12 }

func TestEventStudyByHand(t *testing.T) {
	prices, dates := studyData()
	events := flagTable([]bool{true, true, true, false, false, true}, make([]bool, 6))
	res, err := EventStudy(prices, events, dates, EventStudyOptions{ForwardOptions: ForwardOptions{Horizons: []int{1}}})
	if err != nil {
		t.Fatal(err)
	}
	h := res[0]
	// Events at rows 0, 1 and 2 are bought on the next row: excess 5%, 5%, 0%.
	// The event at row 5 has no row after it: missing.
	if h.Observations != 3 || h.Missing != 1 || !near(h.MeanExcess, 0.10/3) || !near(h.WinRate, 2.0/3) {
		t.Errorf("got %+v", h)
	}
	// January holds the first two events (mean 5%), February the third (0%).
	if h.Months != 2 || !near(h.MonthlyMean, 0.025) || math.IsNaN(h.TStat) {
		t.Errorf("monthly: %+v", h)
	}
}

func TestEventStudyMinGapDelayLevelsAndTradable(t *testing.T) {
	prices, dates := studyData()
	events := flagTable([]bool{true, true, true, false, false, false}, make([]bool, 6))

	gap, err := EventStudy(prices, events, dates, EventStudyOptions{ForwardOptions: ForwardOptions{Horizons: []int{1}}, MinGap: 2})
	if err != nil || gap[0].Observations != 2 {
		t.Errorf("MinGap 2 should keep rows 0 and 2: %+v, %v", gap, err)
	}

	// Delay 2 buys at row 2 for the event at row 0: 110 -> 121 is 10%, the
	// market 5%.
	one := flagTable([]bool{true, false, false, false, false, false}, make([]bool, 6))
	delayed, err := EventStudy(prices, one, dates, EventStudyOptions{ForwardOptions: ForwardOptions{Horizons: []int{1}, Delay: 2}})
	if err != nil || !near(delayed[0].MeanExcess, 0.05) {
		t.Errorf("Delay 2: %+v, %v", delayed, err)
	}

	// A flat benchmark index leaves the raw return.
	flat := insyra.NewDataList(100.0, 100.0, 100.0, 100.0, 100.0, 100.0)
	lv, err := EventStudy(prices, one, dates, EventStudyOptions{ForwardOptions: ForwardOptions{Horizons: []int{1}, Benchmark: BenchmarkLevels, BenchmarkLevels: flat}})
	if err != nil || !near(lv[0].MeanExcess, 0.10) {
		t.Errorf("levels: %+v, %v", lv, err)
	}

	// Not tradable at the entry row: the event is missing.
	mask := flagTable([]bool{true, false, true, true, true, true}, []bool{true, true, true, true, true, true})
	tr, err := EventStudy(prices, one, dates, EventStudyOptions{ForwardOptions: ForwardOptions{Horizons: []int{1}, Tradable: mask}})
	if err != nil || tr[0].Observations != 0 || tr[0].Missing != 1 {
		t.Errorf("tradable: %+v, %v", tr, err)
	}
}

func TestSignalBucketsByHand(t *testing.T) {
	prices, dates := studyData()
	sigA := insyra.NewDataList(1.0, -1.0, 1.0, 1.0, 1.0, 1.0).SetName("A")
	sigB := insyra.NewDataList(-1.0, -1.0, -1.0, -1.0, -1.0, -1.0).SetName("B")
	got, err := SignalBuckets(insyra.NewDataTable(sigA, sigB), prices, dates, BucketOptions{ForwardOptions: ForwardOptions{Horizons: []int{1}}, Edges: []float64{0}})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || !math.IsInf(got[0].Lower, -1) || got[0].Upper != 0 || got[1].Lower != 0 || !math.IsInf(got[1].Upper, 1) {
		t.Fatalf("bucket bounds: %+v", got)
	}
	low, high := got[0].Horizons[0], got[1].Horizons[0]
	if low.Observations != 5 || !near(low.MeanExcess, -0.02) || !near(low.WinRate, 0.2) {
		t.Errorf("low bucket: %+v", low)
	}
	if high.Observations != 3 || high.Missing != 2 || !near(high.MeanExcess, 0.10/3) || !near(high.WinRate, 2.0/3) {
		t.Errorf("high bucket: %+v", high)
	}
}

func TestSignalBucketsQuantilesAndOwnMean(t *testing.T) {
	prices := insyra.NewDataTable(insyra.NewDataList(100.0, 101.0, 103.0, 102.0, 105.0, 104.0, 107.0, 108.0).SetName("index"))
	dates := insyra.NewDataList("2026-01-02", "2026-01-05", "2026-01-06", "2026-01-07", "2026-02-02", "2026-02-03", "2026-02-04", "2026-02-05")
	signal := insyra.NewDataTable(insyra.NewDataList(1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0).SetName("index"))
	got, err := SignalBuckets(signal, prices, dates, BucketOptions{
		ForwardOptions: ForwardOptions{Horizons: []int{1}, Benchmark: BenchmarkOwnMean}, Quantiles: []float64{0.5}})
	if err != nil {
		t.Fatal(err)
	}
	// The median of 1..8 is 4.5: signals 1-4 below it, 5-8 above.
	if got[0].Upper != 4.5 || got[1].Lower != 4.5 {
		t.Errorf("edges: %+v", got)
	}
	// The benchmark is the mean one-row return over all seven windows.
	px := []float64{100, 101, 103, 102, 105, 104, 107, 108}
	var ret []float64
	for k := 0; k+1 < len(px); k++ {
		ret = append(ret, px[k+1]/px[k]-1)
	}
	mean := 0.0
	for _, r := range ret {
		mean += r
	}
	mean /= float64(len(ret))
	// Signals 1-4 sit on rows 0-3 and are entered on rows 1-4; 5-8 on rows 4-7,
	// entered on rows 5-8 where only the first three still have a next row.
	lowWant, highWant := 0.0, 0.0
	for e := 1; e <= 4; e++ {
		lowWant += (ret[e] - mean) / 4
	}
	for e := 5; e <= 6; e++ {
		highWant += (ret[e] - mean) / 2
	}
	if !near(got[0].Horizons[0].MeanExcess, lowWant) || got[0].Horizons[0].Observations != 4 {
		t.Errorf("low bucket %+v, want mean %v", got[0].Horizons[0], lowWant)
	}
	if !near(got[1].Horizons[0].MeanExcess, highWant) || got[1].Horizons[0].Observations != 2 {
		t.Errorf("high bucket %+v, want mean %v", got[1].Horizons[0], highWant)
	}
}

// A random walk has no signal, so no bucket's excess return is significant.
func TestSignalBucketsOnARandomWalkFindNothing(t *testing.T) {
	const rows, stocks = 1200, 30
	rng := newTestRand(5)
	dates := make([]any, rows)
	for i := range dates {
		dates[i] = monthlyDate(i)
	}
	priceCols := make([]*insyra.DataList, stocks)
	signalCols := make([]*insyra.DataList, stocks)
	for s := 0; s < stocks; s++ {
		p := make([]any, rows)
		sg := make([]any, rows)
		level := 100.0
		for i := 0; i < rows; i++ {
			level *= math.Exp(0.01 * rng.NormFloat64())
			p[i] = level
			sg[i] = rng.NormFloat64()
		}
		priceCols[s] = insyra.NewDataList(p...)
		signalCols[s] = insyra.NewDataList(sg...)
	}
	got, err := SignalBuckets(insyra.NewDataTable(signalCols...), insyra.NewDataTable(priceCols...), insyra.NewDataList(dates...),
		BucketOptions{ForwardOptions: ForwardOptions{Horizons: []int{5, 20}}, Edges: []float64{-1, 0, 1}})
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range got {
		for _, h := range b.Horizons {
			if math.Abs(h.TStat) > 3 {
				t.Errorf("bucket %d horizon %d: t = %v on pure noise", b.Index, h.Horizon, h.TStat)
			}
		}
	}
}

func TestForwardStudiesRefuseBadInput(t *testing.T) {
	prices, dates := studyData()
	events := flagTable(make([]bool, 6), make([]bool, 6))
	ok := ForwardOptions{Horizons: []int{1}}
	var nilTable *insyra.DataTable
	cases := map[string]func() error{
		"no horizon": func() error { _, e := EventStudy(prices, events, dates, EventStudyOptions{}); return e },
		"horizon 0": func() error {
			_, e := EventStudy(prices, events, dates, EventStudyOptions{ForwardOptions: ForwardOptions{Horizons: []int{0}}})
			return e
		},
		"delay": func() error {
			_, e := EventStudy(prices, events, dates, EventStudyOptions{ForwardOptions: ForwardOptions{Horizons: []int{1}, Delay: -1}})
			return e
		},
		"gap": func() error {
			_, e := EventStudy(prices, events, dates, EventStudyOptions{ForwardOptions: ok, MinGap: -1})
			return e
		},
		"nil prices": func() error {
			_, e := EventStudy(nilTable, events, dates, EventStudyOptions{ForwardOptions: ok})
			return e
		},
		"nil events": func() error {
			_, e := EventStudy(prices, nilTable, dates, EventStudyOptions{ForwardOptions: ok})
			return e
		},
		"events cols": func() error {
			_, e := EventStudy(prices, insyra.NewDataTable(insyra.NewDataList(make([]any, 6)...)), dates, EventStudyOptions{ForwardOptions: ok})
			return e
		},
		"bad date": func() error {
			_, e := EventStudy(prices, events, insyra.NewDataList(1, 2, 3, 4, 5, 6), EventStudyOptions{ForwardOptions: ok})
			return e
		},
		"levels missing": func() error {
			_, e := EventStudy(prices, events, dates, EventStudyOptions{ForwardOptions: ForwardOptions{Horizons: []int{1}, Benchmark: BenchmarkLevels}})
			return e
		},
		"benchmark": func() error {
			_, e := EventStudy(prices, events, dates, EventStudyOptions{ForwardOptions: ForwardOptions{Horizons: []int{1}, Benchmark: BenchmarkKind(9)}})
			return e
		},
		"no edges": func() error {
			_, e := SignalBuckets(prices, prices, dates, BucketOptions{ForwardOptions: ok})
			return e
		},
		"edge order": func() error {
			_, e := SignalBuckets(prices, prices, dates, BucketOptions{ForwardOptions: ok, Edges: []float64{1, 1}})
			return e
		},
		"quantile": func() error {
			_, e := SignalBuckets(prices, prices, dates, BucketOptions{ForwardOptions: ok, Quantiles: []float64{1}})
			return e
		},
		"bias n":  func() error { _, e := ta.Bias(insyra.NewDataList(1.0), 0); return e },
		"biasz w": func() error { _, e := ta.BiasZ(insyra.NewDataList(1.0), 1, 1); return e },
	}
	for name, f := range cases {
		if f() == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}
