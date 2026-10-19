package quant

import (
	"errors"
	"fmt"
	"math"
	"sort"

	"github.com/HazelnutParadise/insyra"
)

// PivotKind says whether a turning point is a high or a low.
type PivotKind int

const (
	// PivotLow is a trough: the price rose by the threshold after it.
	PivotLow PivotKind = iota
	// PivotHigh is a peak: the price fell by the threshold after it.
	PivotHigh
)

// String returns the constant name of k, or PivotKind(n) for a value that is
// not one of the constants.
func (k PivotKind) String() string {
	switch k {
	case PivotLow:
		return "PivotLow"
	case PivotHigh:
		return "PivotHigh"
	}
	return fmt.Sprintf("PivotKind(%d)", int(k))
}

// Pivot is one turning point found by ZigZag. Index and Price locate the
// extreme; ConfirmedAt is the first row on which the price had moved the
// threshold away from it, the earliest a trader could know about it. Trade on
// ConfirmedAt and describe the series with Index. Segment counts the data
// breaks before the pivot, and is 0 unless ZigZagBy was given Breaks.
type Pivot struct {
	Kind        PivotKind
	Index       int
	Price       float64
	ConfirmedAt int
	Segment     int
}

// ZigZag finds the turning points of a price series that reverse by at least
// threshold, a fraction such as 0.10 for 10%. Prices must be positive and
// finite.
//
// It is causal: a pivot is reported at the row that confirms it and uses no
// later price, so recomputing on a series cut short returns exactly the pivots
// of the full series whose ConfirmedAt lies before the cut. Pivots alternate
// between lows and highs. The extreme still being tracked at the end, which has
// not been confirmed, is not reported. A price that ties the running extreme
// leaves the earlier row as the extreme.
func ZigZag(prices insyra.IDataList, threshold float64) ([]Pivot, error) {
	if math.IsNaN(threshold) || threshold <= 0 || threshold >= 1 {
		return nil, errors.New("threshold must be a fraction between 0 and 1, such as 0.10 for 10%")
	}
	p, err := positivePrices(prices)
	if err != nil {
		return nil, err
	}
	th := make([]float64, len(p))
	for i := range th {
		th[i] = threshold
	}
	return zigzagCore(p, th, nil), nil
}

// ZigZagOptions are the optional settings of ZigZagBy and VolatilityThresholds.
type ZigZagOptions struct {
	// Breaks are the rows on which a new data segment starts, for example the
	// first row after a trading halt, a relisting or a gap in the data. Nothing
	// is carried over a break: the extreme being tracked is dropped, and a
	// pivot is never made from rows on both sides. Row 0 starts the first
	// segment on its own and need not be listed.
	Breaks []int
}

// ZigZagBy is ZigZag with a threshold for every row, so the reversal that
// counts can follow the volatility of the day, for example 3 times the 60-day
// standard deviation (see VolatilityThresholds). thresholds[i] is the fraction
// a reversal from an extreme at row i must reach, so it is known on the day the
// extreme is made and the result stays causal. Each threshold must be a
// fraction between 0 and 1.
//
// Pivots carry the number of the data segment they lie in (see
// ZigZagOptions.Breaks). The pivot still being tracked when a segment ends is
// not reported.
func ZigZagBy(prices, thresholds insyra.IDataList, opts ...ZigZagOptions) ([]Pivot, error) {
	o, err := zigzagOptions(opts)
	if err != nil {
		return nil, err
	}
	p, err := positivePrices(prices)
	if err != nil {
		return nil, err
	}
	th, err := numericSeries(thresholds, "thresholds")
	if err != nil {
		return nil, err
	}
	if len(th) != len(p) {
		return nil, fmt.Errorf("thresholds has %d rows but prices has %d", len(th), len(p))
	}
	for i, v := range th {
		if v <= 0 || v >= 1 {
			return nil, fmt.Errorf("thresholds holds %v at row index %d; a threshold must be a fraction between 0 and 1", v, i)
		}
	}
	breaks, err := cleanBreaks(o.Breaks, len(p))
	if err != nil {
		return nil, err
	}
	return zigzagCore(p, th, breaks), nil
}

func zigzagOptions(opts []ZigZagOptions) (ZigZagOptions, error) {
	if len(opts) > 1 {
		return ZigZagOptions{}, errors.New("at most one ZigZagOptions is accepted")
	}
	if len(opts) == 1 {
		return opts[0], nil
	}
	return ZigZagOptions{}, nil
}

// cleanBreaks returns breaks sorted, without duplicates and without row 0.
func cleanBreaks(breaks []int, n int) ([]int, error) {
	out := make([]int, 0, len(breaks))
	for _, b := range breaks {
		if b < 0 || b >= n {
			return nil, fmt.Errorf("break at row index %d is outside the %d rows", b, n)
		}
		if b > 0 {
			out = append(out, b)
		}
	}
	sort.Ints(out)
	uniq := out[:0]
	for i, b := range out {
		if i == 0 || b != out[i-1] {
			uniq = append(uniq, b)
		}
	}
	return uniq, nil
}

// segmentBounds returns the [start, end) row ranges the breaks cut n rows into.
func segmentBounds(n int, breaks []int) [][2]int {
	var out [][2]int
	start := 0
	for _, b := range breaks {
		out = append(out, [2]int{start, b})
		start = b
	}
	return append(out, [2]int{start, n})
}

func positivePrices(prices insyra.IDataList) ([]float64, error) {
	return positiveSeries(prices, "prices")
}

func positiveSeries(dl insyra.IDataList, label string) ([]float64, error) {
	p, err := numericSeries(dl, label)
	if err != nil {
		return nil, err
	}
	for i, v := range p {
		if v <= 0 {
			return nil, fmt.Errorf("%s holds %v at row index %d; prices must be positive", label, v, i)
		}
	}
	return p, nil
}

func zigzagCore(p, th []float64, breaks []int) []Pivot {
	var pivots []Pivot
	for seg, b := range segmentBounds(len(p), breaks) {
		pivots = zigzagSegment(pivots, p, th, b[0], b[1], seg)
	}
	return pivots
}

func zigzagSegment(pivots []Pivot, p, th []float64, from, to, seg int) []Pivot {
	if to-from < 2 {
		return pivots
	}
	lowIdx, highIdx := from, from
	direction := 0 // 0 undecided, +1 rising since a confirmed low, -1 falling since a confirmed high
	for t := from + 1; t < to; t++ {
		switch direction {
		case 0:
			if p[t] < p[lowIdx] {
				lowIdx = t
			}
			if p[t] > p[highIdx] {
				highIdx = t
			}
			if p[t] >= p[lowIdx]*(1+th[lowIdx]) {
				pivots = append(pivots, Pivot{PivotLow, lowIdx, p[lowIdx], t, seg})
				direction, highIdx = 1, t
			} else if p[t] <= p[highIdx]*(1-th[highIdx]) {
				pivots = append(pivots, Pivot{PivotHigh, highIdx, p[highIdx], t, seg})
				direction, lowIdx = -1, t
			}
		case 1:
			if p[t] > p[highIdx] {
				highIdx = t
			} else if p[t] <= p[highIdx]*(1-th[highIdx]) {
				pivots = append(pivots, Pivot{PivotHigh, highIdx, p[highIdx], t, seg})
				direction, lowIdx = -1, t
			}
		case -1:
			if p[t] < p[lowIdx] {
				lowIdx = t
			} else if p[t] >= p[lowIdx]*(1+th[lowIdx]) {
				pivots = append(pivots, Pivot{PivotLow, lowIdx, p[lowIdx], t, seg})
				direction, highIdx = 1, t
			}
		}
	}
	return pivots
}

// VolatilityOptions are the settings of VolatilityThresholds. A zero field
// takes the default shown.
type VolatilityOptions struct {
	// Multiple is how many standard deviations of the daily log return a
	// reversal must reach. Required, and above 0.
	Multiple float64
	// Window is the number of rows of returns the standard deviation looks at.
	// Default 60.
	Window int
	// MinRows is the fewest returns needed before the standard deviation is
	// trusted. Before that, WarmupVolatility stands in for it. Default 20.
	MinRows int
	// Floor is the smallest threshold returned. Default 0.02.
	Floor float64
	// WarmupVolatility is the daily volatility assumed while there are fewer
	// than MinRows returns. Default 0.025.
	WarmupVolatility float64
	// Breaks are the rows on which a new data segment starts, as in
	// ZigZagOptions. The return across a break is not used, and the window
	// restarts there.
	Breaks []int
}

// VolatilityThresholds returns, for every row, max(Floor, Multiple times the
// standard deviation of the daily log returns over the last Window rows), the
// per-row threshold ZigZagBy takes. It looks only at returns up to the row
// itself, so using it does not look ahead. The standard deviation is the
// population one over the returns available, up to Window of them.
func VolatilityThresholds(prices insyra.IDataList, opts VolatilityOptions) (*insyra.DataList, error) {
	if math.IsNaN(opts.Multiple) || opts.Multiple <= 0 {
		return nil, errors.New("option Multiple must be above 0")
	}
	if opts.Window < 0 || opts.MinRows < 0 || opts.Floor < 0 || opts.WarmupVolatility < 0 {
		return nil, errors.New("options Window, MinRows, Floor and WarmupVolatility must not be negative")
	}
	if opts.Window == 0 {
		opts.Window = 60
	}
	if opts.MinRows == 0 {
		opts.MinRows = 20
	}
	if opts.Floor == 0 {
		opts.Floor = 0.02
	}
	if opts.WarmupVolatility == 0 {
		opts.WarmupVolatility = 0.025
	}
	if opts.MinRows > opts.Window {
		return nil, errors.New("option MinRows must not exceed Window")
	}
	p, err := positivePrices(prices)
	if err != nil {
		return nil, err
	}
	breaks, err := cleanBreaks(opts.Breaks, len(p))
	if err != nil {
		return nil, err
	}
	out := make([]any, len(p))
	for _, b := range segmentBounds(len(p), breaks) {
		for i := b[0]; i < b[1]; i++ {
			k := i - b[0] // returns available: rows b[0]+1 .. i
			if k > opts.Window {
				k = opts.Window
			}
			if k < opts.MinRows {
				out[i] = math.Max(opts.Floor, opts.Multiple*opts.WarmupVolatility)
				continue
			}
			var sum, sumSq float64
			for j := i - k + 1; j <= i; j++ {
				r := math.Log(p[j] / p[j-1])
				sum += r
				sumSq += r * r
			}
			v := sumSq/float64(k) - (sum/float64(k))*(sum/float64(k))
			sd := 0.0
			if v > 0 {
				sd = math.Sqrt(v)
			}
			out[i] = math.Max(opts.Floor, opts.Multiple*sd)
		}
	}
	return insyra.NewDataList(out...), nil
}

// RegimeKind says whether a stretch of the price series is a bull or a bear
// market.
type RegimeKind int

const (
	// RegimeBull runs from a low to the next high.
	RegimeBull RegimeKind = iota
	// RegimeBear runs from a high to the next low.
	RegimeBear
)

// String returns the constant name of k, or RegimeKind(n) for a value that is
// not one of the constants.
func (k RegimeKind) String() string {
	switch k {
	case RegimeBull:
		return "RegimeBull"
	case RegimeBear:
		return "RegimeBear"
	}
	return fmt.Sprintf("RegimeKind(%d)", int(k))
}

// Regime is one bull or bear stretch between two consecutive pivots. Start and
// End are the rows of the two extremes, Length is End - Start, Change is the
// price change from the first to the second as a fraction, and ConfirmedAt is
// the row that confirmed the closing pivot, when the regime could first be
// known to have ended.
type Regime struct {
	Kind        RegimeKind
	Start, End  int
	Length      int
	Change      float64
	ConfirmedAt int
}

// RegimeOptions are the optional settings of Regimes.
type RegimeOptions struct {
	// MinPhase is the fewest rows a bull or bear stretch may last. A shorter
	// one is removed together with the two turning points around it, so the
	// stretches on both sides join. 0 keeps every stretch. This is the minimum
	// phase length of Pagan and Sossounov (2003).
	MinPhase int
	// MinCycle is the fewest rows from one low to the next low, or one high to
	// the next high. A turning point that makes a shorter cycle is removed with
	// the one before it. 0 applies no minimum. This is the minimum cycle length
	// of Pagan and Sossounov (2003).
	MinCycle int
}

// Regimes turns the pivots of ZigZag into bull stretches (low to high) and bear
// stretches (high to low). The stretch before the first pivot and after the
// last are not reported, since neither has both ends. Pivots must alternate, as
// ZigZag returns them.
//
// Removing turning points to enforce MinPhase and MinCycle uses the whole list,
// so unlike ZigZag it is not causal: a regime reported as ending at a pivot may
// be merged into a longer one once later pivots are known. Use ConfirmedAt only
// without minimums.
func Regimes(pivots []Pivot, opts ...RegimeOptions) ([]Regime, error) {
	if len(opts) > 1 {
		return nil, errors.New("Regimes accepts at most one RegimeOptions")
	}
	var o RegimeOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	if o.MinPhase < 0 || o.MinCycle < 0 {
		return nil, errors.New("MinPhase and MinCycle must not be negative")
	}
	for i := 1; i < len(pivots); i++ {
		if pivots[i].Kind == pivots[i-1].Kind {
			return nil, fmt.Errorf("pivots at list positions %d and %d are both %v; they must alternate", i-1, i, pivots[i].Kind)
		}
		if pivots[i].Index <= pivots[i-1].Index {
			return nil, fmt.Errorf("pivot at list position %d is not after the one before it", i)
		}
	}
	kept := append([]Pivot(nil), pivots...)
	for changed := true; changed; {
		changed = false
		if o.MinPhase > 0 {
			for i := 0; i+1 < len(kept); i++ {
				if kept[i+1].Index-kept[i].Index < o.MinPhase {
					kept = append(kept[:i], kept[i+2:]...)
					changed = true
					break
				}
			}
		}
		if !changed && o.MinCycle > 0 {
			for i := 0; i+2 < len(kept); i++ {
				if kept[i+2].Index-kept[i].Index < o.MinCycle {
					kept = append(kept[:i+1], kept[i+3:]...)
					changed = true
					break
				}
			}
		}
	}
	regimes := make([]Regime, 0, len(kept))
	for i := 0; i+1 < len(kept); i++ {
		a, b := kept[i], kept[i+1]
		kind := RegimeBull
		if a.Kind == PivotHigh {
			kind = RegimeBear
		}
		regimes = append(regimes, Regime{kind, a.Index, b.Index, b.Index - a.Index, b.Price/a.Price - 1, b.ConfirmedAt})
	}
	return regimes, nil
}
