package quant

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/isr"
)

type abnormalCase struct {
	Days    []int       `json:"days"`
	MeanAR  []float64   `json:"mean_ar"`
	ARt     []float64   `json:"ar_t"`
	ARp     []float64   `json:"ar_p"`
	MeanCAR []float64   `json:"mean_car"`
	CARt    []float64   `json:"car_t"`
	CARp    []float64   `json:"car_p"`
	Events  int         `json:"events"`
	Missing int         `json:"missing"`
	Kept    [][2]int    `json:"kept"`
	CARs    [][]float64 `json:"cars"`
}

type abnormalBaseline struct {
	Prices           [][]*float64            `json:"prices"`
	Levels           []float64               `json:"levels"`
	Events           map[string][]int        `json:"events"`
	EventWindow      [2]int                  `json:"event_window"`
	EstimationWindow [2]int                  `json:"estimation_window"`
	MinEstimation    int                     `json:"min_estimation"`
	Cases            map[string]abnormalCase `json:"cases"`
}

// abnormalPanel builds the tables of a study from price series, event rows per
// stock and market levels.
func abnormalPanel(prices [][]any, events map[int][]int, levels []float64) (pt, et *insyra.DataTable, dates, lv *insyra.DataList) {
	rows := len(levels)
	var pc, ec []*insyra.DataList
	for i, p := range prices {
		pc = append(pc, insyra.NewDataList(p...).SetName(fmt.Sprintf("S%d", i)))
		flags := make([]any, rows)
		for r := range flags {
			flags[r] = false
		}
		for _, r := range events[i] {
			flags[r] = true
		}
		ec = append(ec, insyra.NewDataList(flags...).SetName(fmt.Sprintf("S%d", i)))
	}
	ds := make([]any, rows)
	lvs := make([]any, rows)
	for r := range ds {
		ds[r] = monthlyDate(r)
		lvs[r] = levels[r]
	}
	return insyra.NewDataTable(pc...), insyra.NewDataTable(ec...), insyra.NewDataList(ds...), insyra.NewDataList(lvs...)
}

func closeTo(t *testing.T, label string, got, want []float64, tol float64) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: %d values, want %d", label, len(got), len(want))
	}
	for i := range got {
		if math.IsNaN(want[i]) != math.IsNaN(got[i]) || (!math.IsNaN(want[i]) && math.Abs(got[i]-want[i]) > tol) {
			t.Errorf("%s[%d] = %v, want %v", label, i, got[i], want[i])
		}
	}
}

// TestAbnormalReturnsMatchesNumpyScipy compares every number with a
// reference written in numpy and scipy (abnormal_returns_baseline.py).
func TestAbnormalReturnsMatchesNumpyScipy(t *testing.T) {
	raw, err := os.ReadFile("testdata/abnormal_returns_baseline.json")
	if err != nil {
		t.Fatal(err)
	}
	var base abnormalBaseline
	if err := json.Unmarshal(raw, &base); err != nil {
		t.Fatal(err)
	}
	prices := make([][]any, len(base.Prices))
	for i, col := range base.Prices {
		for _, v := range col {
			if v == nil {
				prices[i] = append(prices[i], nil)
			} else {
				prices[i] = append(prices[i], *v)
			}
		}
	}
	events := map[int][]int{}
	for k, v := range base.Events {
		var i int
		if _, err := fmt.Sscan(k, &i); err != nil {
			t.Fatal(err)
		}
		events[i] = v
	}
	pt, et, dates, lv := abnormalPanel(prices, events, base.Levels)

	configs := map[string]AbnormalReturnsOptions{
		"market_adjusted": {Model: AbnormalMarketAdjusted},
		"market_model":    {Model: AbnormalMarketModel, EstimationWindow: base.EstimationWindow, MinEstimation: base.MinEstimation},
		"market_model_mingap": {Model: AbnormalMarketModel, EstimationWindow: base.EstimationWindow,
			MinEstimation: base.MinEstimation, MinGap: 50},
	}
	for name, opts := range configs {
		want := base.Cases[name]
		opts.EventWindow = base.EventWindow
		opts.MarketLevels = lv
		res, err := AbnormalReturns(pt, et, dates, opts)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if res.Events != want.Events || res.Missing != want.Missing {
			t.Fatalf("%s: events %d missing %d, want %d and %d", name, res.Events, res.Missing, want.Events, want.Missing)
		}
		for i, d := range want.Days {
			if res.Days[i] != d {
				t.Fatalf("%s: day %d is %d, want %d", name, i, res.Days[i], d)
			}
		}
		const tol = 1e-9
		closeTo(t, name+" MeanAR", res.MeanAR, want.MeanAR, tol)
		closeTo(t, name+" ARTStat", res.ARTStat, want.ARt, tol)
		closeTo(t, name+" ARPValue", res.ARPValue, want.ARp, tol)
		closeTo(t, name+" MeanCAR", res.MeanCAR, want.MeanCAR, tol)
		closeTo(t, name+" CARTStat", res.CARTStat, want.CARt, tol)
		closeTo(t, name+" CARPValue", res.CARPValue, want.CARp, tol)

		rows, cols := res.CARs.Size()
		if rows != want.Events || cols != 2+len(want.Days) {
			t.Fatalf("%s: CARs is %dx%d", name, rows, cols)
		}
		for e, k := range want.Kept {
			if got := res.CARs.GetElementByNumberIndex(e, 0); got != fmt.Sprintf("S%d", k[0]) {
				t.Errorf("%s: CARs stock of row %d is %v", name, e, got)
			}
			if got := res.CARs.GetElementByNumberIndex(e, 1); got != monthlyDate(k[1]) {
				t.Errorf("%s: CARs date of row %d is %v", name, e, got)
			}
			got := make([]float64, len(want.Days))
			for d := range got {
				got[d], _ = insyra.ToFloat64Safe(res.CARs.GetElementByNumberIndex(e, 2+d))
			}
			closeTo(t, fmt.Sprintf("%s CARs[%d]", name, e), got, want.CARs[e], tol)
		}
	}
}

// plantedPanel builds nStocks stocks that follow beta*market plus noise, with
// an event every spacing rows from row start, and adds extra[day] to the
// stock's return around every event.
func plantedPanel(seed int64, nStocks, rows, start, spacing int, extra map[int]float64) (pt, et *insyra.DataTable, dates, lv *insyra.DataList) {
	rng := newTestRand(seed)
	rm := make([]float64, rows)
	levels := make([]float64, rows)
	level := 1000.0
	for t := range rm {
		rm[t] = rng.NormFloat64() * 0.01
		level *= 1 + rm[t]
		levels[t] = level
	}
	prices := make([][]any, nStocks)
	events := map[int][]int{}
	for i := range prices {
		beta := 0.7 + 0.1*float64(i%7)
		r := make([]float64, rows)
		for t := range r {
			r[t] = beta*rm[t] + rng.NormFloat64()*0.01
		}
		for s := start + i*3; s < rows-30; s += spacing {
			events[i] = append(events[i], s)
			for d, v := range extra {
				r[s+d] += v
			}
		}
		p := 50.0
		prices[i] = make([]any, rows)
		for t := range r {
			p *= 1 + r[t]
			prices[i][t] = p
		}
	}
	return abnormalPanel(prices, events, levels)
}

func TestAbnormalReturnsRecoversPlantedEffect(t *testing.T) {
	pt, et, dates, lv := plantedPanel(11, 12, 700, 280, 60, map[int]float64{0: 0.05})
	res, err := AbnormalReturns(pt, et, dates, AbnormalReturnsOptions{
		Model: AbnormalMarketModel, EstimationWindow: [2]int{-250, -11}, EventWindow: [2]int{-10, 20},
		MinEstimation: 100, MarketLevels: lv,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Events < 20 {
		t.Fatalf("only %d events", res.Events)
	}
	d0 := 10 // index of day 0 in {-10..20}
	if res.Days[d0] != 0 {
		t.Fatalf("day index %d is %d", d0, res.Days[d0])
	}
	if math.Abs(res.MeanAR[d0]-0.05) > 0.01 {
		t.Errorf("MeanAR on day 0 = %v, want about 0.05", res.MeanAR[d0])
	}
	if res.ARPValue[d0] > 1e-6 {
		t.Errorf("day 0 p = %v, want significant", res.ARPValue[d0])
	}
	// A quiet day is not significant at 0.001.
	if res.ARPValue[d0-5] < 0.001 {
		t.Errorf("day -5 p = %v", res.ARPValue[d0-5])
	}
	if res.MeanCAR[20] < 0.03 {
		t.Errorf("CAR at day 10 = %v", res.MeanCAR[20])
	}
}

func TestAbnormalReturnsShowsPreEventLeak(t *testing.T) {
	extra := map[int]float64{-5: 0.01, -4: 0.01, -3: 0.01, -2: 0.01, -1: 0.01}
	pt, et, dates, lv := plantedPanel(12, 12, 700, 280, 60, extra)
	res, err := AbnormalReturns(pt, et, dates, AbnormalReturnsOptions{
		Model: AbnormalMarketModel, EstimationWindow: [2]int{-250, -11}, EventWindow: [2]int{-10, 5},
		MinEstimation: 100, MarketLevels: lv,
	})
	if err != nil {
		t.Fatal(err)
	}
	idx := func(day int) int { return day + 10 }
	if res.MeanCAR[idx(-1)]-res.MeanCAR[idx(-6)] < 0.03 {
		t.Errorf("CAR rose only %v over days -5..-1", res.MeanCAR[idx(-1)]-res.MeanCAR[idx(-6)])
	}
	for d := -4; d <= -1; d++ {
		if res.MeanCAR[idx(d)] <= res.MeanCAR[idx(d-1)] {
			t.Errorf("CAR did not rise on day %d", d)
		}
	}
	if res.CARPValue[idx(-1)] > 1e-4 {
		t.Errorf("CAR p on day -1 = %v", res.CARPValue[idx(-1)])
	}
}

// exactPanel is one stock that follows the market exactly (alpha 0, beta 1)
// except on the rows listed in shocks.
func exactPanel(rows int, events []int, shocks map[int]float64) (pt, et *insyra.DataTable, dates, lv *insyra.DataList) {
	rng := newTestRand(5)
	levels := make([]float64, rows)
	prices := make([]any, rows)
	level, p := 1000.0, 50.0
	for t := 0; t < rows; t++ {
		rm := rng.NormFloat64() * 0.01
		level *= 1 + rm
		levels[t] = level
		p *= 1 + rm + shocks[t]
		prices[t] = p
	}
	return abnormalPanel([][]any{prices}, map[int][]int{0: events}, levels)
}

func TestAbnormalReturnsLeavesOutOtherEventsWindowsFromEstimation(t *testing.T) {
	shocks := map[int]float64{}
	for r := 98; r <= 103; r++ {
		shocks[r] = 0.1 // event A's window, 100 +/- (-2, 3)
	}
	shocks[130] = 0.02 // event B's own reaction
	pt, et, dates, lv := exactPanel(200, []int{100, 130}, shocks)
	opts := AbnormalReturnsOptions{
		Model: AbnormalMarketModel, EstimationWindow: [2]int{-50, -11}, EventWindow: [2]int{-2, 3},
		MinEstimation: 10, MarketLevels: lv,
	}
	res, err := AbnormalReturns(pt, et, dates, opts)
	if err != nil {
		t.Fatal(err)
	}
	if res.Events != 2 {
		t.Fatalf("events %d", res.Events)
	}
	// B's estimation window 80..119 holds A's window 98..103; with those rows
	// left out the fit is exact, so B's only abnormal return is its 2% shock.
	for d := range res.Days {
		got, _ := insyra.ToFloat64Safe(res.CARs.GetElementByNumberIndex(1, 2+d))
		want := 0.0
		if res.Days[d] >= 0 {
			want = 0.02
		}
		if math.Abs(got-want) > 1e-9 {
			t.Errorf("B CAR on day %d = %v, want %v", res.Days[d], got, want)
		}
	}
	// 34 rows are usable for B and 40 for A, so 35 drops only B.
	opts.MinEstimation = 35
	res, err = AbnormalReturns(pt, et, dates, opts)
	if err != nil {
		t.Fatal(err)
	}
	if res.Events != 1 || res.Missing != 1 {
		t.Errorf("MinEstimation 35: events %d missing %d, want 1 and 1", res.Events, res.Missing)
	}
}

func TestAbnormalReturnsDropsWholeEvents(t *testing.T) {
	pt, et, dates, lv := plantedPanel(21, 4, 400, 100, 50, map[int]float64{0: 0.04})
	opts := AbnormalReturnsOptions{Model: AbnormalMarketAdjusted, EventWindow: [2]int{-3, 5}, MarketLevels: lv}
	clean, err := AbnormalReturns(pt, et, dates, opts)
	if err != nil {
		t.Fatal(err)
	}
	// Put a missing price on day +4 of the first event of the first stock.
	flags := et.GetColByNumber(0).Data()
	first := -1
	for r, v := range flags {
		if v == true {
			first = r
			break
		}
	}
	pt.UpdateElement(first+4, "A", nil)
	dropped, err := AbnormalReturns(pt, et, dates, opts)
	if err != nil {
		t.Fatal(err)
	}
	if dropped.Events != clean.Events-1 || dropped.Missing != clean.Missing+1 {
		t.Fatalf("events %d missing %d, was %d and %d", dropped.Events, dropped.Missing, clean.Events, clean.Missing)
	}
	// Every day uses the same events: with that event unflagged, and the gap
	// still in the data, nothing differs.
	et.UpdateElement(first, "A", false)
	again, err := AbnormalReturns(pt, et, dates, opts)
	if err != nil {
		t.Fatal(err)
	}
	if again.Events != dropped.Events || again.Missing != dropped.Missing-1 {
		t.Errorf("unflagged: events %d missing %d; with a gap: %d and %d", again.Events, again.Missing, dropped.Events, dropped.Missing)
	}
	closeTo(t, "MeanAR", dropped.MeanAR, again.MeanAR, 1e-12)
	closeTo(t, "MeanCAR", dropped.MeanCAR, again.MeanCAR, 1e-12)
	closeTo(t, "ARTStat", dropped.ARTStat, again.ARTStat, 1e-12)
	closeTo(t, "CARPValue", dropped.CARPValue, again.CARPValue, 1e-12)
	rows, _ := dropped.CARs.Size()
	if rows != dropped.Events {
		t.Errorf("CARs has %d rows for %d events", rows, dropped.Events)
	}
}

func TestAbnormalReturnsDropsEventsOutsideTheTable(t *testing.T) {
	pt, et, dates, lv := exactPanel(100, []int{0, 2, 50, 97, 99}, nil)
	res, err := AbnormalReturns(pt, et, dates, AbnormalReturnsOptions{
		EventWindow: [2]int{-2, 3}, MarketLevels: lv,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Row 2 needs the return at row 0, which has no previous price; 97 and 99 run off the end.
	if res.Events != 1 || res.Missing != 4 {
		t.Errorf("events %d missing %d, want 1 and 4", res.Events, res.Missing)
	}
}

func TestAbnormalReturnsDropsEventsWithMissingMarket(t *testing.T) {
	pt, et, dates, lv := exactPanel(100, []int{30, 60}, nil)
	lv.Update(32, math.NaN())
	res, err := AbnormalReturns(pt, et, dates, AbnormalReturnsOptions{EventWindow: [2]int{-2, 3}, MarketLevels: lv})
	if err != nil {
		t.Fatal(err)
	}
	if res.Events != 1 || res.Missing != 1 {
		t.Errorf("events %d missing %d, want 1 and 1", res.Events, res.Missing)
	}
}

func TestAbnormalReturnsMinGap(t *testing.T) {
	pt, et, dates, lv := exactPanel(200, []int{50, 55, 100}, nil)
	opts := AbnormalReturnsOptions{EventWindow: [2]int{-1, 1}, MarketLevels: lv}
	all, err := AbnormalReturns(pt, et, dates, opts)
	if err != nil {
		t.Fatal(err)
	}
	opts.MinGap = 10
	gap, err := AbnormalReturns(pt, et, dates, opts)
	if err != nil {
		t.Fatal(err)
	}
	if all.Events != 3 || gap.Events != 2 || gap.Missing != 0 {
		t.Errorf("events %d, with MinGap %d (missing %d)", all.Events, gap.Events, gap.Missing)
	}
}

func TestAbnormalReturnsOneEventHasNoTStatistic(t *testing.T) {
	pt, et, dates, lv := exactPanel(100, []int{50}, map[int]float64{50: 0.03})
	res, err := AbnormalReturns(pt, et, dates, AbnormalReturnsOptions{EventWindow: [2]int{-1, 1}, MarketLevels: lv})
	if err != nil {
		t.Fatal(err)
	}
	if res.Events != 1 || math.Abs(res.MeanAR[1]-0.03) > 1e-9 {
		t.Fatalf("events %d, MeanAR %v", res.Events, res.MeanAR)
	}
	for d := range res.Days {
		if !math.IsNaN(res.ARTStat[d]) || !math.IsNaN(res.ARPValue[d]) || !math.IsNaN(res.CARTStat[d]) || !math.IsNaN(res.CARPValue[d]) {
			t.Errorf("day %d has a statistic with one event", res.Days[d])
		}
	}
}

func TestAbnormalReturnsNoEvents(t *testing.T) {
	pt, et, dates, lv := exactPanel(100, nil, nil)
	res, err := AbnormalReturns(pt, et, dates, AbnormalReturnsOptions{EventWindow: [2]int{-1, 1}, MarketLevels: lv})
	if err != nil {
		t.Fatal(err)
	}
	if res.Events != 0 || !math.IsNaN(res.MeanAR[0]) {
		t.Errorf("events %d, MeanAR %v", res.Events, res.MeanAR)
	}
	if rows, cols := res.CARs.Size(); rows != 0 || cols != 5 {
		t.Errorf("CARs is %dx%d", rows, cols)
	}
}

func TestAbnormalReturnsRejectsBadInput(t *testing.T) {
	pt, et, dates, lv := exactPanel(100, []int{50}, nil)
	good := AbnormalReturnsOptions{Model: AbnormalMarketModel, EstimationWindow: [2]int{-30, -11}, EventWindow: [2]int{-5, 5},
		MinEstimation: 10, MarketLevels: lv}
	with := func(f func(o *AbnormalReturnsOptions)) AbnormalReturnsOptions {
		o := good
		f(&o)
		return o
	}
	shortLevels := insyra.NewDataList(1.0, 2.0)
	badLevels := insyra.NewDataList(lv.Data()...)
	badLevels.Update(10, -5.0)
	badEvents := insyra.NewDataTable(insyra.NewDataList(make([]any, 100)...).SetName("S0"))
	badEvents.UpdateElement(3, "A", "yes")
	twoCols := insyra.NewDataTable(et.GetColByNumber(0), et.GetColByNumber(0))
	wrongRows := insyra.NewDataList(dates.Data()[:50]...)
	text := insyra.NewDataTable(insyra.NewDataList("a", "b").SetName("S0"))

	cases := []struct {
		name string
		call func() error
		want string
	}{
		{"nil prices", func() error { _, e := AbnormalReturns(nil, et, dates, good); return e }, "prices is nil"},
		{"nil events", func() error { _, e := AbnormalReturns(pt, nil, dates, good); return e }, "events is nil"},
		{"nil dates", func() error { _, e := AbnormalReturns(pt, et, nil, good); return e }, "dates is nil"},
		{"nil levels", func() error {
			_, e := AbnormalReturns(pt, et, dates, with(func(o *AbnormalReturnsOptions) { o.MarketLevels = nil }))
			return e
		}, "MarketLevels is nil"},
		{"levels too short", func() error {
			_, e := AbnormalReturns(pt, et, dates, with(func(o *AbnormalReturnsOptions) { o.MarketLevels = shortLevels }))
			return e
		}, "MarketLevels has 2 rows"},
		{"non-positive level", func() error {
			_, e := AbnormalReturns(pt, et, dates, with(func(o *AbnormalReturnsOptions) { o.MarketLevels = badLevels }))
			return e
		}, "must be positive"},
		{"events not boolean", func() error { _, e := AbnormalReturns(pt, badEvents, dates, good); return e }, "events column 0"},
		{"event columns differ", func() error { _, e := AbnormalReturns(pt, twoCols, dates, good); return e }, "events has 2 columns"},
		{"dates length", func() error { _, e := AbnormalReturns(pt, et, wrongRows, good); return e }, "price column 0 has 100 rows"},
		{"text prices", func() error { _, e := AbnormalReturns(text, et, insyra.NewDataList(1, 2), good); return e }, "non-numeric"},
		{"unsupported model", func() error {
			_, e := AbnormalReturns(pt, et, dates, with(func(o *AbnormalReturnsOptions) { o.Model = AbnormalModel(7) }))
			return e
		}, "AbnormalModel(7)"},
		{"zero MinEstimation", func() error {
			_, e := AbnormalReturns(pt, et, dates, with(func(o *AbnormalReturnsOptions) { o.MinEstimation = 0 }))
			return e
		}, "MinEstimation 0 must be at least 3"},
		{"negative MinEstimation", func() error {
			_, e := AbnormalReturns(pt, et, dates, with(func(o *AbnormalReturnsOptions) { o.MinEstimation = -4 }))
			return e
		}, "MinEstimation -4"},
		{"MinEstimation too large", func() error {
			_, e := AbnormalReturns(pt, et, dates, with(func(o *AbnormalReturnsOptions) { o.MinEstimation = 21 }))
			return e
		}, "more than the 20 rows"},
		{"estimation overlaps event window", func() error {
			_, e := AbnormalReturns(pt, et, dates, with(func(o *AbnormalReturnsOptions) { o.EstimationWindow = [2]int{-30, -5} }))
			return e
		}, "must end before EventWindow"},
		{"estimation reversed", func() error {
			_, e := AbnormalReturns(pt, et, dates, with(func(o *AbnormalReturnsOptions) { o.EstimationWindow = [2]int{-11, -30} }))
			return e
		}, "starts after it ends"},
		{"event window reversed", func() error {
			_, e := AbnormalReturns(pt, et, dates, with(func(o *AbnormalReturnsOptions) { o.EventWindow = [2]int{5, -5} }))
			return e
		}, "EventWindow [5 -5] starts after it ends"},
		{"adjusted with estimation window", func() error {
			_, e := AbnormalReturns(pt, et, dates, with(func(o *AbnormalReturnsOptions) { o.Model = AbnormalMarketAdjusted }))
			return e
		}, "must be zero for AbnormalMarketAdjusted"},
		{"adjusted with MinEstimation", func() error {
			_, e := AbnormalReturns(pt, et, dates, AbnormalReturnsOptions{EventWindow: [2]int{-1, 1}, MinEstimation: 5, MarketLevels: lv})
			return e
		}, "MinEstimation 5 must be 0"},
		{"negative MinGap", func() error {
			_, e := AbnormalReturns(pt, et, dates, with(func(o *AbnormalReturnsOptions) { o.MinGap = -1 }))
			return e
		}, "MinGap must not be negative"},
	}
	for _, c := range cases {
		err := c.call()
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: error %v does not contain %q", c.name, err, c.want)
		}
	}
	if _, err := AbnormalReturns(pt, et, dates, good); err != nil {
		t.Errorf("the good options fail: %v", err)
	}
}

func TestAbnormalReturnsAcceptsIsrWrappers(t *testing.T) {
	pt, et, dates, lv := plantedPanel(31, 6, 400, 100, 50, map[int]float64{0: 0.03})
	opts := AbnormalReturnsOptions{Model: AbnormalMarketModel, EstimationWindow: [2]int{-60, -11}, EventWindow: [2]int{-3, 5},
		MinEstimation: 30, MarketLevels: lv}
	want, err := AbnormalReturns(pt, et, dates, opts)
	if err != nil {
		t.Fatal(err)
	}
	opts.MarketLevels = isr.UseDL(lv)
	got, err := AbnormalReturns(isr.UseDT(pt), isr.UseDT(et), isr.UseDL(dates), opts)
	if err != nil {
		t.Fatal(err)
	}
	if got.Events != want.Events || got.Events == 0 {
		t.Fatalf("events %d, want %d", got.Events, want.Events)
	}
	closeTo(t, "MeanAR", got.MeanAR, want.MeanAR, 0)
	closeTo(t, "CARPValue", got.CARPValue, want.CARPValue, 0)
}

// A date cell that is a slice stays one cell: the table has one row per event.
func TestAbnormalReturnsKeepsASliceDateAsOneCell(t *testing.T) {
	pt, et, dates, lv := exactPanel(100, []int{40, 70}, nil)
	dates.Update(40, insyra.Cell([]int{2026, 1, 2}))
	res, err := AbnormalReturns(pt, et, dates, AbnormalReturnsOptions{EventWindow: [2]int{-1, 1}, MarketLevels: lv})
	if err != nil {
		t.Fatal(err)
	}
	rows, cols := res.CARs.Size()
	if res.Events != 2 || rows != 2 || cols != 5 {
		t.Fatalf("events %d, CARs %dx%d, want 2 events and 2x5", res.Events, rows, cols)
	}
	got, ok := res.CARs.GetElementByNumberIndex(0, 1).([]int)
	if !ok || len(got) != 3 || got[0] != 2026 {
		t.Errorf("event date of row 0 is %v", res.CARs.GetElementByNumberIndex(0, 1))
	}
	if g := res.CARs.GetElementByNumberIndex(1, 1); g != monthlyDate(70) {
		t.Errorf("event date of row 1 is %v", g)
	}
}

// A ratio of two finite prices can overflow; that return is missing data, so
// the event is dropped as a whole instead of carrying an infinity.
func TestAbnormalReturnsTreatsAnOverflowingReturnAsMissing(t *testing.T) {
	pt, et, dates, lv := exactPanel(100, []int{30, 70}, nil)
	pt.UpdateElement(31, "A", 1e-300)
	pt.UpdateElement(32, "A", 1e300) // 1e300/1e-300 overflows
	res, err := AbnormalReturns(pt, et, dates, AbnormalReturnsOptions{EventWindow: [2]int{-2, 3}, MarketLevels: lv})
	if err != nil {
		t.Fatal(err)
	}
	if res.Events != 1 || res.Missing != 1 {
		t.Fatalf("events %d missing %d, want 1 and 1", res.Events, res.Missing)
	}
	for d := range res.Days {
		if math.IsInf(res.MeanAR[d], 0) || math.IsInf(res.MeanCAR[d], 0) {
			t.Errorf("day %d is infinite", res.Days[d])
		}
	}
	// The same holds for the market.
	pt, et, dates, lv = exactPanel(100, []int{30, 70}, nil)
	lv.Update(31, 1e-300)
	lv.Update(32, 1e300)
	res, err = AbnormalReturns(pt, et, dates, AbnormalReturnsOptions{EventWindow: [2]int{-2, 3}, MarketLevels: lv})
	if err != nil {
		t.Fatal(err)
	}
	if res.Events != 1 || res.Missing != 1 {
		t.Errorf("market: events %d missing %d, want 1 and 1", res.Events, res.Missing)
	}
}
