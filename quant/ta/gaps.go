package ta

import (
	"errors"
	"fmt"
	"math"

	"github.com/HazelnutParadise/insyra"
)

// GapKind is the class of a price gap.
type GapKind int

const (
	// GapCommon is a gap inside the range of the last LookbackRows rows, which
	// the price usually fills soon.
	GapCommon GapKind = iota
	// GapBreakaway is a gap that takes the price out of its range to a new
	// extreme after a stretch without a clear trend.
	GapBreakaway
	// GapRunaway is a gap to a new extreme in the middle of a trend that was
	// already running in the same direction.
	GapRunaway
	// GapExhaustion is a gap to a new extreme after a long trend, on heavy
	// volume.
	GapExhaustion
	// GapUnclassified is a gap none of the above describes, for example a gap
	// up to a new high straight after a sharp fall.
	GapUnclassified
)

// String returns the constant name of k, or GapKind(n) for a value that is not
// one of the constants.
func (k GapKind) String() string {
	switch k {
	case GapCommon:
		return "GapCommon"
	case GapBreakaway:
		return "GapBreakaway"
	case GapRunaway:
		return "GapRunaway"
	case GapExhaustion:
		return "GapExhaustion"
	case GapUnclassified:
		return "GapUnclassified"
	}
	return fmt.Sprintf("GapKind(%d)", int(k))
}

// GapFill says whether a gap was closed within a number of rows.
type GapFill int

const (
	// GapFillUnknown means the data ends, or a new segment starts, before the
	// window is over.
	GapFillUnknown GapFill = iota
	// GapFilled means the price came back to the edge of the gap in the window.
	GapFilled
	// GapNotFilled means it did not.
	GapNotFilled
)

// String returns the constant name of f, or GapFill(n) for a value that is not
// one of the constants.
func (f GapFill) String() string {
	switch f {
	case GapFillUnknown:
		return "GapFillUnknown"
	case GapFilled:
		return "GapFilled"
	case GapNotFilled:
		return "GapNotFilled"
	}
	return fmt.Sprintf("GapFill(%d)", int(f))
}

// Gap is one price gap and its class.
//
// Row is the row of the gap day; Direction is BreakUp for a gap up (its low is
// above the previous high) and BreakDown for a gap down (its high is below the
// previous low). RelativeVolume is the day's volume over the mean of the LookbackRows
// before it (NaN when that mean is 0), TrendZ the move over those rows in
// units of the volatility to be expected over them, and HighVolume whether
// RelativeVolume reached HighVolumeRatio. Kind, RelativeVolume, TrendZ and
// HighVolume use only rows up to the gap day, so they can be used to trade it.
//
// Fills holds, for each of GapOptions.FillWindows in the same order, whether the
// gap was filled within that many rows after the gap day. It looks at the rows
// after the gap day, so it is what happened afterwards, an outcome to study and
// not something to trade on. A gap up is filled when a later low reaches the previous high, a gap down
// when a later high reaches the previous low.
type Gap struct {
	Row            int
	Direction      BreakDirection // BreakUp or BreakDown
	Kind           GapKind
	HighVolume     bool
	RelativeVolume float64
	TrendZ         float64
	Fills          []GapFill // one per GapOptions.FillWindows
}

// GapOptions are the optional settings of ClassifyGaps. A zero field takes the
// default shown.
type GapOptions struct {
	// SegmentStarts are the rows on which a new data segment starts, as in
	// ZigZagOptions. A gap is not looked for until a segment has enough rows,
	// and the fill windows stop at the end of the segment.
	SegmentStarts []int
	// LookbackRows is the number of rows whose range, mean volume and price move
	// a gap is judged against. Default 20.
	LookbackRows int
	// VolatilityRows is the number of rows of daily log returns the volatility
	// in TrendZ is measured over. Default 60.
	VolatilityRows int
	// HighVolumeRatio is the relative volume from which a day counts as heavy.
	// Default 1.5.
	HighVolumeRatio float64
	// RunawayZ is the size of the earlier move, in standard deviations, from
	// which a gap to a new extreme is a runaway and not a breakaway. Default 1.
	RunawayZ float64
	// FillWindows are the numbers of rows after the gap day within which a fill
	// is looked for, each giving one entry of Gap.Fills. Default 5 and 20. Each
	// must be at least 1.
	FillWindows []int
	// ExhaustionZ is the size of the earlier move from which a gap to a new
	// extreme on heavy volume is an exhaustion gap. Default 2.
	ExhaustionZ float64
}

// ClassifyGaps finds the price gaps in a series of daily bars and classifies
// them as common, breakaway, runaway or exhaustion gaps.
//
// A gap up has a low above the previous high and a gap down a high below the
// previous low. It is judged against the LookbackRows rows before the gap day:
//
//   - Common: the close stays inside the range of those rows.
//   - Breakaway: the close is a new extreme (above the highest high for a gap
//     up, below the lowest low for a gap down) after a move of less than
//     RunawayZ standard deviations in either direction.
//   - Runaway: a new extreme after a move of between RunawayZ and ExhaustionZ
//     standard deviations in the gap's direction.
//   - Exhaustion: a new extreme after a move of ExhaustionZ standard deviations
//     or more in the gap's direction, on volume of HighVolumeRatio times the
//     recent mean or more.
//
// The move is the return over the LookbackRows rows ending the day before,
// divided by the standard deviation of daily log returns over VolatilityRows
// rows times the square root of LookbackRows. Everything in the class uses
// only rows up to the gap day. A gap is only looked for from row
// max(LookbackRows+1, VolatilityRows)+1 of its segment, row 61 with the defaults. Gaps that fit none of
// the classes are returned as GapUnclassified.
func ClassifyGaps(high, low, close, volume insyra.IDataList, opts ...GapOptions) ([]Gap, error) {
	if len(opts) > 1 {
		return nil, errors.New("ClassifyGaps accepts at most one GapOptions")
	}
	var o GapOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	if o.LookbackRows < 0 || o.VolatilityRows < 0 || o.HighVolumeRatio < 0 || o.RunawayZ < 0 || o.ExhaustionZ < 0 {
		return nil, errors.New("GapOptions fields must not be negative")
	}
	if o.LookbackRows == 0 {
		o.LookbackRows = 20
	}
	if o.VolatilityRows == 0 {
		o.VolatilityRows = 60
	}
	if o.HighVolumeRatio == 0 {
		o.HighVolumeRatio = 1.5
	}
	if o.RunawayZ == 0 {
		o.RunawayZ = 1
	}
	if o.FillWindows == nil {
		o.FillWindows = []int{5, 20}
	}
	for _, w := range o.FillWindows {
		if w < 1 {
			return nil, fmt.Errorf("FillWindows must be at least 1, got %d", w)
		}
	}
	if o.ExhaustionZ == 0 {
		o.ExhaustionZ = 2
	}
	if o.ExhaustionZ < o.RunawayZ {
		return nil, errors.New("ExhaustionZ must not be below RunawayZ")
	}
	h, err := positiveSeries(high, "high")
	if err != nil {
		return nil, err
	}
	l, err := positiveSeries(low, "low")
	if err != nil {
		return nil, err
	}
	c, err := positivePrices(close)
	if err != nil {
		return nil, err
	}
	m, err := numericSeries(volume, "volume")
	if err != nil {
		return nil, err
	}
	n := len(c)
	if len(h) != n || len(l) != n || len(m) != n {
		return nil, fmt.Errorf("high, low, close and volume must have the same length, got %d, %d, %d and %d", len(h), len(l), n, len(m))
	}
	for i := range c {
		if h[i] < l[i] {
			return nil, fmt.Errorf("high %v is below low %v at row index %d", h[i], l[i], i)
		}
		if m[i] < 0 {
			return nil, fmt.Errorf("volume holds %v at row index %d; it must not be negative", m[i], i)
		}
	}
	starts, err := cleanSegmentStarts(o.SegmentStarts, n)
	if err != nil {
		return nil, err
	}
	segStart := segmentStartOf(n, starts)
	segEnd := make([]int, n) // first row after the row's segment
	for _, b := range segmentBounds(n, starts) {
		for i := b[0]; i < b[1]; i++ {
			segEnd[i] = b[1]
		}
	}
	warmup := max(o.LookbackRows+1, o.VolatilityRows) + 1
	minVolRows := min(20, o.VolatilityRows)

	var out []Gap
	for t := 1; t < n; t++ {
		if t-segStart[t] < warmup {
			continue
		}
		up := l[t] > h[t-1]
		down := h[t] < l[t-1]
		if !up && !down {
			continue
		}
		dir := BreakDown
		if up {
			dir = BreakUp
		}
		hi, lo, meanVol := math.Inf(-1), math.Inf(1), 0.0
		for j := t - o.LookbackRows; j < t; j++ {
			hi = math.Max(hi, h[j])
			lo = math.Min(lo, l[j])
			meanVol += m[j]
		}
		meanVol /= float64(o.LookbackRows)
		rv := math.NaN()
		if meanVol > 0 {
			rv = m[t] / meanVol
		}
		k := min(o.VolatilityRows, t-segStart[t])
		var vol float64
		if k >= minVolRows {
			var sum, sumSq float64
			for j := t - k + 1; j <= t; j++ {
				r := math.Log(c[j] / c[j-1])
				sum += r
				sumSq += r * r
			}
			v := sumSq/float64(k) - (sum/float64(k))*(sum/float64(k))
			if v > 0 {
				vol = math.Sqrt(v)
			}
		}
		z := (c[t-1]/c[t-1-o.LookbackRows] - 1) / (vol * math.Sqrt(float64(o.LookbackRows)))
		newHigh, newLow := c[t] > hi, c[t] < lo
		heavy := rv >= o.HighVolumeRatio

		g := Gap{Row: t, Direction: dir, RelativeVolume: rv, TrendZ: z, HighVolume: heavy, Kind: GapUnclassified}
		extreme, move := newHigh, z
		if !up {
			extreme, move = newLow, -z
		}
		switch {
		case !newHigh && !newLow:
			g.Kind = GapCommon
		case extreme && math.Abs(z) < o.RunawayZ:
			g.Kind = GapBreakaway
		case extreme && move >= o.RunawayZ && move < o.ExhaustionZ:
			g.Kind = GapRunaway
		case extreme && move >= o.ExhaustionZ && heavy:
			g.Kind = GapExhaustion
		}
		edge := h[t-1]
		if !up {
			edge = l[t-1]
		}
		g.Fills = make([]GapFill, len(o.FillWindows))
		for i, w := range o.FillWindows {
			g.Fills[i] = gapFill(h, l, t, w, segEnd[t], up, edge)
		}
		out = append(out, g)
	}
	return out, nil
}

func gapFill(h, l []float64, t, window, end int, up bool, edge float64) GapFill {
	if t+window >= end {
		return GapFillUnknown
	}
	for j := t + 1; j <= t+window; j++ {
		if (up && l[j] <= edge) || (!up && h[j] >= edge) {
			return GapFilled
		}
	}
	return GapNotFilled
}
