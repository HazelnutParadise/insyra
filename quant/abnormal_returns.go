package quant

import (
	"errors"
	"fmt"
	"math"

	"github.com/HazelnutParadise/insyra"
	"gonum.org/v1/gonum/stat/distuv"
)

// AbnormalModel chooses what a stock is expected to return on an event day.
type AbnormalModel int

const (
	// AbnormalMarketAdjusted expects the stock to return what the market
	// returned that day. Nothing is estimated.
	AbnormalMarketAdjusted AbnormalModel = iota
	// AbnormalMarketModel expects alpha + beta * market return, with alpha and
	// beta fitted by ordinary least squares on the stock's own estimation
	// window.
	AbnormalMarketModel
)

// AbnormalReturnsOptions are the settings of AbnormalReturns.
type AbnormalReturnsOptions struct {
	// Model chooses the expected return. The default is AbnormalMarketAdjusted.
	Model AbnormalModel
	// EstimationWindow is the first and last row, relative to the event row,
	// used to fit the market model, for example {-250, -11}. It is required for
	// AbnormalMarketModel, must end before EventWindow starts, and must be the
	// zero value for AbnormalMarketAdjusted.
	EstimationWindow [2]int
	// EventWindow is the first and last row, relative to the event row, that an
	// abnormal return is reported for, for example {-10, 20}. Row 0 is the row
	// the event is flagged on. The first must not be after the last.
	EventWindow [2]int
	// MinEstimation is the fewest usable estimation rows an event needs under
	// AbnormalMarketModel; an event with fewer is dropped. It has no default and
	// must be at least 3. It must be 0 for AbnormalMarketAdjusted.
	MinEstimation int
	// MarketLevels are the market index levels, one per row, required for both
	// models. Returns are taken as level / previous level - 1, as for prices.
	// A NaN level is a missing one.
	MarketLevels insyra.IDataList
	// MinGap is the fewest rows between two events of the same stock that are
	// both counted: an event closer than that to the last counted one is
	// dropped. 0 counts every event.
	MinGap int
}

// AbnormalReturnsResult is the abnormal return around the events, day by day.
type AbnormalReturnsResult struct {
	// Days are the relative day numbers of EventWindow; the event row is 0.
	Days []int
	// MeanAR and MeanCAR are the average abnormal return and the average
	// cumulative abnormal return (cumulated from the first day of EventWindow)
	// of the events on each day.
	MeanAR, MeanCAR []float64
	// ARTStat and ARPValue test, for each day, that the mean abnormal return is
	// zero across events: t = mean / (sd / sqrt(N)) with N-1 degrees of freedom
	// and a two-tailed p. CARTStat and CARPValue do the same for the CAR. All
	// four are NaN when fewer than 2 events were used or the events agree
	// exactly (zero spread).
	ARTStat, ARPValue   []float64
	CARTStat, CARPValue []float64
	// Events is how many events were used and Missing how many were left out
	// for lack of data; every day uses the same events.
	Events, Missing int
	// CARs has one row per used event: the stock (its column name in prices),
	// the event date as given in dates, and the CAR on each relative day.
	CARs *insyra.DataTable
}

// AbnormalReturns shows how a stock moved against its normal around each
// event. For every true cell of events, it estimates how the stock normally
// moves with the market (from a window before the event, or by taking the
// market's own return), then reports for each day from before to after the
// event how far the stock was above or below that normal, and the running
// total of those gaps (the CAR). The days before the event show whether the
// news leaked early.
//
// prices has one column per stock and one row per date, events has the same
// shape with true where the event happened at that row's close, and dates gives
// each row's date. dates is only used to check the number of rows and to fill
// the event-date column of CARs; it may hold any values. Rows must be in date
// order. Returns are simple returns, p[t]/p[t-1]-1.
//
// An event is dropped, and counted in Missing, as a whole when the stock's
// price or the market level is missing on any row the event window needs
// (including the row before the first one), when the event window runs outside
// the table, or, for AbnormalMarketModel, when fewer than MinEstimation usable
// estimation rows remain or the market did not move at all on them. Rows of a
// stock's estimation window that fall inside the event window of another event
// of the same stock are not used, since that event's own reaction would
// distort the fit.
func AbnormalReturns(prices, events insyra.IDataTable, dates insyra.IDataList, opts AbnormalReturnsOptions) (*AbnormalReturnsResult, error) {
	if err := checkAbnormalOptions(opts); err != nil {
		return nil, err
	}
	if insyra.IsNilTable(prices) {
		return nil, errors.New("prices is nil")
	}
	if insyra.IsNilTable(events) {
		return nil, errors.New("events is nil")
	}
	if insyra.IsNilList(dates) {
		return nil, errors.New("dates is nil")
	}
	if insyra.IsNilList(opts.MarketLevels) {
		return nil, errors.New("MarketLevels is nil; both models need the market")
	}

	var snap *abnormalInput
	var err error
	// AtomicDoAll locks only the core types, so wrappers such as isr's are
	// unwrapped to the core object they hand to AtomicDo.
	var lockPrices, lockEvents, lockDates, lockLevels any
	prices.AtomicDo(func(t *insyra.DataTable) {
		if t != nil {
			lockPrices = t
		}
	})
	events.AtomicDo(func(t *insyra.DataTable) {
		if t != nil {
			lockEvents = t
		}
	})
	dates.AtomicDo(func(l *insyra.DataList) {
		if l != nil {
			lockDates = l
		}
	})
	opts.MarketLevels.AtomicDo(func(l *insyra.DataList) {
		if l != nil {
			lockLevels = l
		}
	})
	insyra.AtomicDoAll(func() {
		snap, err = readAbnormalInput(prices, events, dates, opts.MarketLevels, opts.EventWindow)
	}, lockPrices, lockEvents, lockDates, lockLevels)
	if err != nil {
		return nil, err
	}
	return computeAbnormalReturns(snap, opts), nil
}

// maxAbnormalOffset bounds every window end and the number of rows, so that
// no sum of a row and an offset, or a window's length, can overflow an int,
// even on 32 bits.
const maxAbnormalOffset = 1 << 29

func checkAbnormalOptions(o AbnormalReturnsOptions) error {
	windows := []struct {
		name string
		v    [2]int
	}{{"EstimationWindow", o.EstimationWindow}, {"EventWindow", o.EventWindow}}
	for _, w := range windows {
		for _, x := range w.v {
			if x < -maxAbnormalOffset || x > maxAbnormalOffset {
				return fmt.Errorf("%s %v is out of range; each end must be within %d rows of the event", w.name, w.v, maxAbnormalOffset)
			}
		}
	}
	switch o.Model {
	case AbnormalMarketAdjusted:
		if o.EstimationWindow != [2]int{} {
			return fmt.Errorf("EstimationWindow %v must be zero for %v, which estimates nothing", o.EstimationWindow, o.Model)
		}
		if o.MinEstimation != 0 {
			return fmt.Errorf("MinEstimation %d must be 0 for %v, which estimates nothing", o.MinEstimation, o.Model)
		}
	case AbnormalMarketModel:
		if o.MinEstimation < 3 {
			return fmt.Errorf("MinEstimation %d must be at least 3 for %v; it has no default", o.MinEstimation, o.Model)
		}
		if o.EstimationWindow[0] > o.EstimationWindow[1] {
			return fmt.Errorf("EstimationWindow %v starts after it ends", o.EstimationWindow)
		}
		if o.EstimationWindow[1] >= o.EventWindow[0] {
			return fmt.Errorf("EstimationWindow %v must end before EventWindow %v starts", o.EstimationWindow, o.EventWindow)
		}
		if size := o.EstimationWindow[1] - o.EstimationWindow[0] + 1; o.MinEstimation > size {
			return fmt.Errorf("MinEstimation %d is more than the %d rows of EstimationWindow %v", o.MinEstimation, size, o.EstimationWindow)
		}
	default:
		return fmt.Errorf("unsupported model %v", o.Model)
	}
	if o.EventWindow[0] > o.EventWindow[1] {
		return fmt.Errorf("EventWindow %v starts after it ends", o.EventWindow)
	}
	if o.MinGap < 0 {
		return errors.New("MinGap must not be negative")
	}
	return nil
}

// abnormalInput is the snapshot of the inputs taken under the locks.
type abnormalInput struct {
	rows   int
	names  []string
	prices [][]float64
	flags  [][]bool
	dates  []any
	levels []float64
}

func readAbnormalInput(prices, events insyra.IDataTable, dates insyra.IDataList, levels insyra.IDataList, eventWindow [2]int) (*abnormalInput, error) {
	in := &abnormalInput{}
	dates.AtomicDo(func(l *insyra.DataList) { in.dates = l.Data() })
	in.rows = len(in.dates)
	if in.rows > maxAbnormalOffset {
		return nil, fmt.Errorf("dates has %d rows; at most %d are supported", in.rows, maxAbnormalOffset)
	}
	// Checked before the prices, events and levels are copied. The window ends
	// were bounded by checkAbnormalOptions, so this cannot overflow.
	if span := eventWindow[1] - eventWindow[0] + 1; span > in.rows {
		return nil, fmt.Errorf("EventWindow %v spans %d rows and the table has %d; no event could fit", eventWindow, span, in.rows)
	}
	var err error
	if in.prices, in.names, err = readPriceColumns(prices, in.rows); err != nil {
		return nil, err
	}
	var cols [][]any
	events.AtomicDo(func(dt *insyra.DataTable) {
		for c := 0; c < dt.NumCols(); c++ {
			cols = append(cols, dt.GetColByNumber(c).Data())
		}
	})
	if len(cols) != len(in.prices) {
		return nil, fmt.Errorf("events has %d columns and prices has %d", len(cols), len(in.prices))
	}
	in.flags = make([][]bool, len(cols))
	for c, col := range cols {
		if in.flags[c], err = boolColumn(col, "events", c, in.rows); err != nil {
			return nil, err
		}
	}
	if in.levels, err = numericSeriesAllowNaN(levels, "MarketLevels"); err != nil {
		return nil, err
	}
	if len(in.levels) != in.rows {
		return nil, fmt.Errorf("MarketLevels has %d rows and dates has %d", len(in.levels), in.rows)
	}
	for i, v := range in.levels {
		if !math.IsNaN(v) && v <= 0 {
			return nil, fmt.Errorf("MarketLevels holds %v at row index %d; levels must be positive", v, i)
		}
	}
	return in, nil
}

// simpleReturns returns r[t] = x[t]/x[t-1]-1, NaN for t = 0 and wherever
// either level is missing or the ratio overflows (a return that is not a
// finite number is missing data).
func simpleReturns(x []float64) []float64 {
	r := make([]float64, len(x))
	for t := range x {
		r[t] = math.NaN()
		if t > 0 && !math.IsNaN(x[t]) && !math.IsNaN(x[t-1]) {
			if v := x[t]/x[t-1] - 1; !math.IsInf(v, 0) {
				r[t] = v
			}
		}
	}
	return r
}

type abnormalEvent struct {
	stock, row int
	car        []float64
}

func computeAbnormalReturns(in *abnormalInput, o AbnormalReturnsOptions) *AbnormalReturnsResult {
	w0, w1 := o.EventWindow[0], o.EventWindow[1]
	nd := w1 - w0 + 1
	rm := simpleReturns(in.levels)

	var kept []abnormalEvent
	missing := 0
	ar := make([][]float64, nd) // ar[day][event]
	cars := make([][]float64, nd)

	for stock, flags := range in.flags {
		rs := simpleReturns(in.prices[stock])
		// Every flagged event contaminates, including one MinGap drops.
		var flagged []int
		for r, hit := range flags {
			if hit {
				flagged = append(flagged, r)
			}
		}
		last := -1
		for _, s := range flagged {
			if o.MinGap > 0 && last >= 0 && s-last < o.MinGap {
				continue
			}
			last = s
			day, ok := eventAbnormal(rs, rm, flagged, s, o)
			if !ok {
				missing++
				continue
			}
			car := make([]float64, nd)
			sum := 0.0
			finite := true
			for d, v := range day {
				sum += v
				if math.IsNaN(sum) || math.IsInf(sum, 0) {
					finite = false
					break
				}
				car[d] = sum
			}
			if !finite { // the running total overflowed: missing, like a non-finite return
				missing++
				continue
			}
			for d, v := range day {
				ar[d] = append(ar[d], v)
				cars[d] = append(cars[d], car[d])
			}
			kept = append(kept, abnormalEvent{stock: stock, row: s, car: car})
		}
	}

	res := &AbnormalReturnsResult{Events: len(kept), Missing: missing}
	res.Days = make([]int, nd)
	res.MeanAR, res.MeanCAR = make([]float64, nd), make([]float64, nd)
	res.ARTStat, res.ARPValue = make([]float64, nd), make([]float64, nd)
	res.CARTStat, res.CARPValue = make([]float64, nd), make([]float64, nd)
	for d := range nd {
		res.Days[d] = w0 + d
		res.MeanAR[d], res.ARTStat[d], res.ARPValue[d] = crossSectionalT(ar[d])
		res.MeanCAR[d], res.CARTStat[d], res.CARPValue[d] = crossSectionalT(cars[d])
	}

	table := make([]*insyra.DataList, 0, nd+2)
	stockCol := make([]any, len(kept))
	dateCol := make([]any, len(kept))
	for i, k := range kept {
		stockCol[i] = in.names[k.stock]
		dateCol[i] = insyra.Cell(in.dates[k.row]) // one cell, even for a slice
	}
	table = append(table, insyra.NewDataList(stockCol...).SetName("stock"), insyra.NewDataList(dateCol...).SetName("event date"))
	for d := range nd {
		col := make([]any, len(kept))
		for i, k := range kept {
			col[i] = k.car[d]
		}
		table = append(table, insyra.NewDataList(col...).SetName(fmt.Sprintf("CAR[%d]", res.Days[d])))
	}
	res.CARs = insyra.NewDataTable(table...)
	return res
}

// eventAbnormal returns the abnormal return on every day of the event window
// of the event at row s, or false when the event has to be dropped.
func eventAbnormal(rs, rm []float64, flagged []int, s int, o AbnormalReturnsOptions) ([]float64, bool) {
	w0, w1 := o.EventWindow[0], o.EventWindow[1]
	rows := len(rs)
	if s+w0 < 1 || s+w1 >= rows {
		return nil, false
	}
	for t := s + w0; t <= s+w1; t++ {
		if math.IsNaN(rs[t]) || math.IsNaN(rm[t]) {
			return nil, false
		}
	}
	alpha, beta := 0.0, 1.0
	if o.Model == AbnormalMarketModel {
		var ok bool
		if alpha, beta, ok = fitMarketModel(rs, rm, flagged, s, o); !ok {
			return nil, false
		}
	}
	out := make([]float64, w1-w0+1)
	for d := range out {
		t := s + w0 + d
		out[d] = rs[t] - (alpha + beta*rm[t])
		if math.IsNaN(out[d]) || math.IsInf(out[d], 0) {
			return nil, false
		}
	}
	return out, true
}

// fitMarketModel fits rs = alpha + beta * rm by least squares on the usable
// rows of the estimation window of the event at row s.
func fitMarketModel(rs, rm []float64, flagged []int, s int, o AbnormalReturnsOptions) (alpha, beta float64, ok bool) {
	var xs, ys []float64
	// Only rows 1..len-1 have a return, so the loop is bounded by the table
	// whatever the window's length.
	for t := max(s+o.EstimationWindow[0], 1); t <= min(s+o.EstimationWindow[1], len(rs)-1); t++ {
		if math.IsNaN(rs[t]) || math.IsNaN(rm[t]) {
			continue
		}
		contaminated := false
		for _, e := range flagged {
			if e != s && t >= e+o.EventWindow[0] && t <= e+o.EventWindow[1] {
				contaminated = true
				break
			}
		}
		if contaminated {
			continue
		}
		xs = append(xs, rm[t])
		ys = append(ys, rs[t])
	}
	if len(xs) < o.MinEstimation {
		return 0, 0, false
	}
	n := float64(len(xs))
	mx, my := 0.0, 0.0
	for i := range xs {
		mx += xs[i]
		my += ys[i]
	}
	mx /= n
	my /= n
	sxx, sxy := 0.0, 0.0
	for i := range xs {
		sxx += (xs[i] - mx) * (xs[i] - mx)
		sxy += (xs[i] - mx) * (ys[i] - my)
	}
	if sxx == 0 {
		return 0, 0, false
	}
	beta = sxy / sxx
	alpha = my - beta*mx
	if math.IsNaN(alpha) || math.IsInf(alpha, 0) || math.IsNaN(beta) || math.IsInf(beta, 0) {
		return 0, 0, false
	}
	return alpha, beta, true
}

// crossSectionalT returns the mean of x and the t statistic and two-tailed p
// that the mean is zero, with len(x)-1 degrees of freedom. The statistic and p
// are NaN with fewer than 2 values or no spread; the mean is NaN with none.
func crossSectionalT(x []float64) (mean, t, p float64) {
	n := len(x)
	if n == 0 {
		return math.NaN(), math.NaN(), math.NaN()
	}
	for _, v := range x {
		mean += v
	}
	mean /= float64(n)
	if math.IsNaN(mean) || math.IsInf(mean, 0) { // the sum overflowed
		return math.NaN(), math.NaN(), math.NaN()
	}
	if n < 2 {
		return mean, math.NaN(), math.NaN()
	}
	ss := 0.0
	for _, v := range x {
		ss += (v - mean) * (v - mean)
	}
	sd := math.Sqrt(ss / float64(n-1))
	if math.IsNaN(sd) || math.IsInf(sd, 0) || sd == 0 {
		return mean, math.NaN(), math.NaN()
	}
	t = mean / (sd / math.Sqrt(float64(n)))
	p = 2 * distuv.StudentsT{Mu: 0, Sigma: 1, Nu: float64(n - 1)}.CDF(-math.Abs(t))
	return mean, t, p
}
