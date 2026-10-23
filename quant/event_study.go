package quant

import (
	"errors"
	"fmt"
	"math"
	"sort"

	"github.com/HazelnutParadise/insyra"
)

// EventStudyOptions are the settings of EventStudy.
type EventStudyOptions struct {
	ForwardOptions
	// MinGap is the fewest rows between two events of the same stock that are
	// both counted: an event closer than that to the last counted one (one with a
	// usable return at some horizon) is dropped. 0 counts every event.
	MinGap int
}

// EventStudy measures what happened after each event. For every true cell of
// events, the stock is bought at the close Delay rows after the event row and
// held for each horizon, and its return is taken relative to the benchmark over
// the same window. Events are averaged within calendar months before the t-test,
// so many events on one day are not treated as independent samples.
//
// prices has one column per stock and one row per date, events is a table of the
// same shape with true where the event happened at that row's close, and dates
// gives each row's date as a time.Time or a "YYYY-MM-DD" string. Rows must be in
// date order. An event whose window has a missing price, runs past the last row
// or is not tradable at the entry is counted in Missing and left out. To study a
// sub-period, pass the matching slice of every table and of dates.
//
// It returns one ForwardSummary per horizon, in the order of Horizons.
func EventStudy(prices, events insyra.IDataTable, dates insyra.IDataList, opts EventStudyOptions) ([]ForwardSummary, error) {
	if opts.MinGap < 0 {
		return nil, errors.New("MinGap must not be negative")
	}
	p, err := newPanel(prices, dates, opts.ForwardOptions)
	if err != nil {
		return nil, err
	}
	if insyra.IsNilTable(events) {
		return nil, errors.New("events is nil")
	}
	var cols [][]any
	events.AtomicDo(func(dt *insyra.DataTable) {
		for c := 0; c < dt.NumCols(); c++ {
			cols = append(cols, dt.GetColByNumber(c).Data())
		}
	})
	if len(cols) != len(p.prices) {
		return nil, fmt.Errorf("events has %d columns and prices has %d", len(cols), len(p.prices))
	}

	acc := make([]*accumulator, len(p.opts.Horizons))
	for i := range acc {
		acc[i] = newAccumulator()
	}
	for stock, col := range cols {
		flags, err := boolColumn(col, "events", stock, p.rows)
		if err != nil {
			return nil, err
		}
		last := -1
		for s, hit := range flags {
			if !hit {
				continue
			}
			if opts.MinGap > 0 && last >= 0 && s-last < opts.MinGap {
				continue
			}
			counted := false
			for hi := range acc {
				if v, ok := p.excess(stock, s, hi); ok {
					acc[hi].add(p.month[s], v)
					counted = true
				} else {
					acc[hi].missing++
				}
			}
			// An event with no usable return at any horizon is not counted, so
			// it does not hold back the next one.
			if counted {
				last = s
			}
		}
	}
	return finishSummaries(p, acc)
}

func finishSummaries(p *forwardPanel, acc []*accumulator) ([]ForwardSummary, error) {
	out := make([]ForwardSummary, len(acc))
	for hi, a := range acc {
		s, err := a.summary(p.opts.Horizons[hi], p.opts.NeweyWestLag, p.opts.NoNeweyWest)
		if err != nil {
			return nil, err
		}
		out[hi] = s
	}
	return out, nil
}

// BucketOptions are the settings of SignalBuckets.
type BucketOptions struct {
	ForwardOptions
	// Edges are the bucket boundaries in ascending order. n edges make n+1
	// buckets: the first is below Edges[0], the last from Edges[n-1] up, and the
	// bucket between two edges includes the lower edge.
	Edges []float64
	// Quantiles, used when Edges is empty, are fractions in (0, 1) in ascending
	// order. The edges are those quantiles of every finite signal value in the
	// table, pooled over all dates and stocks.
	Quantiles []float64
}

// BucketSummary is the forward excess return of the observations in one bucket.
type BucketSummary struct {
	Index        int     // 0 is the lowest bucket
	Lower, Upper float64 // the bucket holds Lower <= signal < Upper; -Inf and +Inf at the ends
	Horizons     []ForwardSummary
}

// SignalBuckets sorts every observation of a signal into buckets and reports
// the forward excess return of each bucket, which shows how far a signal has
// to go before the price turns. Every non-missing cell of the signal table is
// an observation; it is traded at the close Delay rows after its row, as in
// EventStudy, and the monthly Newey-West t-test is the same.
//
// signal and prices share a shape: one column per stock, one row per date.
// With one column and Benchmark set to BenchmarkOwnMean this is the version for
// a single series such as a market index, measured against that series' own mean
// return over the same horizon.
func SignalBuckets(signal, prices insyra.IDataTable, dates insyra.IDataList, opts BucketOptions) ([]BucketSummary, error) {
	p, err := newPanel(prices, dates, opts.ForwardOptions)
	if err != nil {
		return nil, err
	}
	if insyra.IsNilTable(signal) {
		return nil, errors.New("signal is nil")
	}
	var cols [][]any
	signal.AtomicDo(func(dt *insyra.DataTable) {
		for c := 0; c < dt.NumCols(); c++ {
			cols = append(cols, dt.GetColByNumber(c).Data())
		}
	})
	if len(cols) != len(p.prices) {
		return nil, fmt.Errorf("signal has %d columns and prices has %d", len(cols), len(p.prices))
	}
	sig := make([][]float64, len(cols))
	for c, col := range cols {
		s, err := finiteOrMissing(col, fmt.Sprintf("signal column %d", c), p.rows)
		if err != nil {
			return nil, err
		}
		sig[c] = s
	}

	edges := append([]float64(nil), opts.Edges...)
	if len(edges) == 0 {
		if len(opts.Quantiles) == 0 {
			return nil, errors.New("edges or quantiles are needed")
		}
		var pooled []float64
		for _, s := range sig {
			for _, v := range s {
				if !math.IsNaN(v) {
					pooled = append(pooled, v)
				}
			}
		}
		if len(pooled) == 0 {
			return nil, errors.New("the signal has no values")
		}
		sort.Float64s(pooled)
		for _, q := range opts.Quantiles {
			if math.IsNaN(q) || q <= 0 || q >= 1 {
				return nil, fmt.Errorf("quantile %v must be between 0 and 1", q)
			}
			pos := q * float64(len(pooled)-1)
			lo := int(math.Floor(pos))
			hi := min(lo+1, len(pooled)-1)
			edges = append(edges, pooled[lo]*(1-(pos-float64(lo)))+pooled[hi]*(pos-float64(lo)))
		}
	}
	for i, e := range edges {
		if math.IsNaN(e) || math.IsInf(e, 0) {
			return nil, fmt.Errorf("edge %d is not finite", i)
		}
		if i > 0 && e <= edges[i-1] {
			return nil, errors.New("edges must be strictly ascending")
		}
	}

	nb := len(edges) + 1
	acc := make([][]*accumulator, nb)
	for b := range acc {
		acc[b] = make([]*accumulator, len(p.opts.Horizons))
		for hi := range acc[b] {
			acc[b][hi] = newAccumulator()
		}
	}
	for stock, s := range sig {
		for row, v := range s {
			if math.IsNaN(v) {
				continue
			}
			b := sort.SearchFloat64s(edges, v)
			if b < len(edges) && edges[b] == v {
				b++ // the bucket above an edge includes the edge
			}
			for hi := range p.opts.Horizons {
				if x, ok := p.excess(stock, row, hi); ok {
					acc[b][hi].add(p.month[row], x)
				} else {
					acc[b][hi].missing++
				}
			}
		}
	}

	out := make([]BucketSummary, nb)
	for b := range out {
		lower, upper := math.Inf(-1), math.Inf(1)
		if b > 0 {
			lower = edges[b-1]
		}
		if b < len(edges) {
			upper = edges[b]
		}
		sums, err := finishSummaries(p, acc[b])
		if err != nil {
			return nil, err
		}
		out[b] = BucketSummary{Index: b, Lower: lower, Upper: upper, Horizons: sums}
	}
	return out, nil
}
