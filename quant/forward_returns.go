package quant

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

// BenchmarkKind chooses what forward returns are measured against.
type BenchmarkKind int

const (
	// BenchmarkCrossSection compares each return with the equal-weighted mean
	// return of every stock that has a price at both ends of the same window
	// (and is tradable, when a mask is given).
	BenchmarkCrossSection BenchmarkKind = iota
	// BenchmarkLevels compares each return with the change of a benchmark index
	// over the same window; the levels are ForwardOptions.BenchmarkLevels.
	BenchmarkLevels
	// BenchmarkOwnMean compares each return with the mean of that series' own
	// returns over the same horizon, for a single series such as a market index
	// where there is no cross-section to average.
	BenchmarkOwnMean
)

// String returns the constant name of k, or BenchmarkKind(n) for a value that
// is not one of the constants.
func (k BenchmarkKind) String() string {
	switch k {
	case BenchmarkCrossSection:
		return "BenchmarkCrossSection"
	case BenchmarkLevels:
		return "BenchmarkLevels"
	case BenchmarkOwnMean:
		return "BenchmarkOwnMean"
	}
	return fmt.Sprintf("BenchmarkKind(%d)", int(k))
}

// ForwardOptions are the settings shared by EventStudy and SignalBuckets.
type ForwardOptions struct {
	// Horizons are the holding periods in rows, each at least 1. At least one
	// is required.
	Horizons []int
	// Delay is the number of rows between the signal row and the entry. The
	// position is entered at the close of row signal+Delay and held for the
	// horizon, so a signal is never traded at the close that produced it. It
	// must be at least 1 and defaults to 1.
	Delay int
	// Benchmark chooses the comparison. The default is BenchmarkCrossSection.
	Benchmark BenchmarkKind
	// BenchmarkLevels are the index levels, one per row, for BenchmarkLevels.
	BenchmarkLevels insyra.IDataList
	// Tradable has the shape of the price table; an observation counts only
	// where it is true at the entry row. Nil means every observation counts.
	Tradable insyra.IDataTable
	// NeweyWestLag is the lag of the t-test on the monthly series. 0 chooses
	// stats.NeweyWestLag(months), a positive value is used as it is, and a
	// negative value is an error. A positive lag must be smaller than the
	// number of months with an observation; EventStudy and SignalBuckets return
	// an error otherwise, instead of shortening it. The automatic lag is capped
	// at months-1.
	NeweyWestLag int
	// NoNeweyWest switches the Newey-West correction off: the t-test then uses
	// lag 0, whose variance divides by the number of months T. That equals
	// linearmodels cov_type="unadjusted" with debiased=False, and is not
	// stats.SingleSampleTTest, which divides by T-1. It cannot be combined with
	// a non-zero NeweyWestLag.
	NoNeweyWest bool
}

// ForwardSummary is the forward excess return for one horizon.
type ForwardSummary struct {
	Horizon int
	// Observations is how many observations had a usable return; Missing is how
	// many were left out because a price was missing, the horizon ran past the
	// last row, or the entry was not tradable.
	Observations, Missing int
	// MeanExcess is the mean excess return over the observations and WinRate the
	// share that beat the benchmark.
	MeanExcess, WinRate float64
	// Months is the number of calendar months with an observation. MonthlyMean,
	// TStat, PValue and Lag describe the Newey-West test that the mean of the
	// monthly averages is zero, which does not treat the many observations of
	// one day as independent. They are NaN with fewer than 2 months.
	Months      int
	MonthlyMean float64
	TStat       float64
	PValue      float64
	Lag         int
}

// forwardPanel is the data of a forward-return study.
type forwardPanel struct {
	prices   [][]float64 // prices[stock][row], NaN where missing
	rows     int
	month    []string // calendar month of each row
	tradable [][]bool // tradable[stock][row] or nil
	opts     ForwardOptions

	crossMean map[[2]int]float64 // (horizon index, entry row) -> benchmark return
	crossDone map[[2]int]bool
	ownMean   map[[2]int]float64 // (horizon index, stock) -> mean return
	levels    []float64
}

// monthOf turns a date cell into a calendar month label.
func monthOf(v any) (string, bool) {
	switch d := v.(type) {
	case time.Time:
		return d.Format("2006-01"), true
	case string:
		if len(d) >= 7 && d[4] == '-' {
			return d[:7], true
		}
	}
	return "", false
}

// newPanel reads the price table, dates and options of a study.
func newPanel(prices insyra.IDataTable, dates insyra.IDataList, o ForwardOptions) (*forwardPanel, error) {
	if len(o.Horizons) == 0 {
		return nil, errors.New("at least one horizon is needed")
	}
	for _, h := range o.Horizons {
		if h < 1 {
			return nil, fmt.Errorf("horizon %d must be at least 1", h)
		}
	}
	if o.NeweyWestLag < 0 {
		return nil, fmt.Errorf("NeweyWestLag %d is negative; 0 chooses it automatically, and NoNeweyWest switches the correction off", o.NeweyWestLag)
	}
	if o.NoNeweyWest && o.NeweyWestLag != 0 {
		return nil, fmt.Errorf("NoNeweyWest cannot be combined with NeweyWestLag %d", o.NeweyWestLag)
	}
	if o.Delay == 0 {
		o.Delay = 1
	}
	if o.Delay < 1 {
		return nil, errors.New("the delay must be at least 1, so a signal is never traded at its own close")
	}
	switch o.Benchmark {
	case BenchmarkCrossSection, BenchmarkLevels, BenchmarkOwnMean:
	default:
		return nil, fmt.Errorf("unsupported benchmark %v", o.Benchmark)
	}
	if insyra.IsNilTable(prices) {
		return nil, errors.New("prices is nil")
	}
	if insyra.IsNilList(dates) {
		return nil, errors.New("dates is nil")
	}

	p := &forwardPanel{opts: o, crossMean: map[[2]int]float64{}, crossDone: map[[2]int]bool{}, ownMean: map[[2]int]float64{}}
	var rawDates []any
	dates.AtomicDo(func(l *insyra.DataList) { rawDates = l.Data() })
	p.rows = len(rawDates)
	p.month = make([]string, p.rows)
	for i, d := range rawDates {
		m, ok := monthOf(d)
		if !ok {
			return nil, fmt.Errorf("dates holds %v at row index %d; use a time.Time or a \"YYYY-MM-DD\" string", d, i)
		}
		p.month[i] = m
	}

	var cols [][]any
	prices.AtomicDo(func(dt *insyra.DataTable) {
		for c := 0; c < dt.NumCols(); c++ {
			cols = append(cols, dt.GetColByNumber(c).Data())
		}
	})
	if len(cols) == 0 {
		return nil, errors.New("prices has no columns")
	}
	p.prices = make([][]float64, len(cols))
	for c, col := range cols {
		series, err := priceColumn(col, c, p.rows)
		if err != nil {
			return nil, err
		}
		p.prices[c] = series
	}

	if o.Tradable != nil {
		if insyra.IsNilTable(o.Tradable) {
			return nil, errors.New("the tradable mask is nil")
		}
		var mask [][]any
		o.Tradable.AtomicDo(func(dt *insyra.DataTable) {
			for c := 0; c < dt.NumCols(); c++ {
				mask = append(mask, dt.GetColByNumber(c).Data())
			}
		})
		if len(mask) != len(cols) {
			return nil, fmt.Errorf("the tradable mask has %d columns and prices has %d", len(mask), len(cols))
		}
		p.tradable = make([][]bool, len(mask))
		for c, col := range mask {
			flags, err := boolColumn(col, "Tradable", c, p.rows)
			if err != nil {
				return nil, err
			}
			p.tradable[c] = flags
		}
	}

	if o.Benchmark == BenchmarkLevels {
		lv, err := numericSeries(o.BenchmarkLevels, "BenchmarkLevels")
		if err != nil {
			return nil, err
		}
		if len(lv) != p.rows {
			return nil, fmt.Errorf("BenchmarkLevels has %d rows and dates has %d", len(lv), p.rows)
		}
		for i, v := range lv {
			if v <= 0 {
				return nil, fmt.Errorf("BenchmarkLevels holds %v at row index %d; levels must be positive", v, i)
			}
		}
		p.levels = lv
	}
	return p, nil
}

func priceColumn(col []any, c, rows int) ([]float64, error) {
	if len(col) != rows {
		return nil, fmt.Errorf("price column %d has %d rows and dates has %d", c, len(col), rows)
	}
	out := make([]float64, rows)
	for i, v := range col {
		if v == nil {
			out[i] = math.NaN()
			continue
		}
		f, ok := insyra.ToFloat64Safe(v)
		if !ok {
			return nil, fmt.Errorf("price column %d contains a non-numeric value at row index %d: %v", c, i, v)
		}
		if math.IsInf(f, 0) || (!math.IsNaN(f) && f <= 0) {
			return nil, fmt.Errorf("price column %d holds %v at row index %d; prices must be positive", c, f, i)
		}
		out[i] = f
	}
	return out, nil
}

func boolColumn(col []any, label string, c, rows int) ([]bool, error) {
	if len(col) != rows {
		return nil, fmt.Errorf("%s column %d has %d rows and dates has %d", label, c, len(col), rows)
	}
	out := make([]bool, rows)
	for i, v := range col {
		switch x := v.(type) {
		case nil:
		case bool:
			out[i] = x
		default:
			f, ok := insyra.ToFloat64Safe(v)
			if !ok || (f != 0 && f != 1) {
				return nil, fmt.Errorf("%s column %d holds %v at row index %d; it must be true, false, 1 or 0", label, c, v, i)
			}
			out[i] = f == 1
		}
	}
	return out, nil
}

// window returns the simple return of stock i from row e to row e+h.
func (p *forwardPanel) window(i, e, h int) (float64, bool) {
	if e+h >= p.rows {
		return 0, false
	}
	a, b := p.prices[i][e], p.prices[i][e+h]
	if math.IsNaN(a) || math.IsNaN(b) {
		return 0, false
	}
	return b/a - 1, true
}

func (p *forwardPanel) isTradable(i, e int) bool {
	return p.tradable == nil || p.tradable[i][e]
}

// benchmark returns the benchmark return of the window from row e over the
// horizon with index hi.
func (p *forwardPanel) benchmark(stock, hi, e int) (float64, bool) {
	h := p.opts.Horizons[hi]
	switch p.opts.Benchmark {
	case BenchmarkLevels:
		if e+h >= p.rows {
			return 0, false
		}
		return p.levels[e+h]/p.levels[e] - 1, true
	case BenchmarkOwnMean:
		key := [2]int{hi, stock}
		if m, ok := p.ownMean[key]; ok {
			return m, !math.IsNaN(m)
		}
		sum, n := 0.0, 0
		for r := 0; r < p.rows; r++ {
			if !p.isTradable(stock, r) {
				continue
			}
			if v, ok := p.window(stock, r, h); ok {
				sum += v
				n++
			}
		}
		m := math.NaN()
		if n > 0 {
			m = sum / float64(n)
		}
		p.ownMean[key] = m
		return m, n > 0
	}
	key := [2]int{hi, e}
	if p.crossDone[key] {
		m := p.crossMean[key]
		return m, !math.IsNaN(m)
	}
	sum, n := 0.0, 0
	for j := range p.prices {
		if !p.isTradable(j, e) {
			continue
		}
		if v, ok := p.window(j, e, h); ok {
			sum += v
			n++
		}
	}
	m := math.NaN()
	if n > 0 {
		m = sum / float64(n)
	}
	p.crossMean[key], p.crossDone[key] = m, true
	return m, n > 0
}

// excess returns the excess return of stock i signalled on row s over horizon
// index hi, and false when it cannot be computed.
func (p *forwardPanel) excess(i, s, hi int) (float64, bool) {
	e := s + p.opts.Delay
	if e >= p.rows || !p.isTradable(i, e) {
		return 0, false
	}
	r, ok := p.window(i, e, p.opts.Horizons[hi])
	if !ok {
		return 0, false
	}
	b, ok := p.benchmark(i, hi, e)
	if !ok {
		return 0, false
	}
	return r - b, true
}

// accumulator collects the excess returns of one horizon.
type accumulator struct {
	n, missing int
	sum        float64
	wins       int
	byMonth    map[string][2]float64 // month -> sum, count
	monthOrder []string
}

func newAccumulator() *accumulator {
	return &accumulator{byMonth: map[string][2]float64{}}
}

func (a *accumulator) add(month string, excess float64) {
	a.n++
	a.sum += excess
	if excess > 0 {
		a.wins++
	}
	sc, seen := a.byMonth[month]
	if !seen {
		a.monthOrder = append(a.monthOrder, month)
	}
	a.byMonth[month] = [2]float64{sc[0] + excess, sc[1] + 1}
}

// summary finishes a horizon: the plain statistics and the Newey-West t-test on
// the monthly means.
func (a *accumulator) summary(horizon, lag int, noNeweyWest bool) (ForwardSummary, error) {
	s := ForwardSummary{Horizon: horizon, Observations: a.n, Missing: a.missing, Months: len(a.monthOrder),
		MeanExcess: math.NaN(), WinRate: math.NaN(), MonthlyMean: math.NaN(), TStat: math.NaN(), PValue: math.NaN()}
	if a.n == 0 {
		return s, nil
	}
	s.MeanExcess = a.sum / float64(a.n)
	s.WinRate = float64(a.wins) / float64(a.n)
	months := append([]string(nil), a.monthOrder...)
	sort.Strings(months)
	monthly := make([]any, len(months))
	total := 0.0
	for i, m := range months {
		v := a.byMonth[m]
		monthly[i] = v[0] / v[1]
		total += v[0] / v[1]
	}
	s.MonthlyMean = total / float64(len(months))
	if len(months) < 2 {
		return s, nil
	}
	useLag := lag
	switch {
	case noNeweyWest:
		useLag = 0
	case lag == 0:
		// Automatic is not a request for a particular lag, so it is capped.
		useLag = min(stats.NeweyWestLag(len(months)), len(months)-1)
	}
	res, err := stats.NeweyWestMeanTest(insyra.NewDataList(monthly...), useLag)
	if err != nil {
		return s, err
	}
	s.TStat, s.PValue, s.Lag = res.Statistic, res.PValue, res.Lag
	return s, nil
}
