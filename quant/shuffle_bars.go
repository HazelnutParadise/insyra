package quant

import (
	"errors"
	"fmt"

	"github.com/HazelnutParadise/insyra"
)

// ShuffleOptions are the optional settings of ShuffleBars.
type ShuffleOptions struct {
	// Seed initializes the random stream and always applies: the same bars and
	// seed give the same path, and the zero value is simply seed 0. Pass a
	// different seed for every path you draw. The stream is a PCG generator, so
	// results do not depend on the Go release.
	Seed uint64
	// SegmentStarts are the rows on which a new data segment starts, as in
	// ZigZagOptions. Bars are shuffled within their own segment and each segment
	// is rebuilt from its own first bar, so no bar crosses a break.
	SegmentStarts []int
}

// ShuffleBars builds a random-walk copy of a series of daily bars, the null to
// compare a price pattern against: how often would this shape appear, or break
// the way the textbook says, if the order of the days meant nothing?
//
// Each bar after the first of its segment is cut into the pieces that do not
// depend on the price level: the gap from the previous close to its open
// (open over the previous close), the high and the low over its open, the close
// over its open, and its volume. The bars are put in a random order and the
// path is rebuilt from the first bar, which stays where it is: each bar's open
// is the rebuilt previous close times its gap, and its high, low and close
// follow from the open by the same ratios. A bar keeps its own shape, so a high
// is still at or above the open and close, and the volume stays with its bar.
// What is destroyed is the order of the days, with the autocorrelation,
// volatility clustering and trend that depend on it. Because the ratios multiply
// to the same total in any order, each segment ends on the same close as the
// real one, so the rebuilt path is a bridge between the real first and last
// closes of its segment.
//
// Prices must be positive, and the high at or above the low. volume may be nil,
// and then the result has no Volume column. The result is a table with the
// columns Open, High, Low, Close and, unless volume is nil, Volume.
func ShuffleBars(open, high, low, close, volume insyra.IDataList, opts ...ShuffleOptions) (*insyra.DataTable, error) {
	if len(opts) > 1 {
		return nil, errors.New("ShuffleBars accepts at most one ShuffleOptions")
	}
	var o ShuffleOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	op, err := positiveSeries(open, "open")
	if err != nil {
		return nil, err
	}
	hi, err := positiveSeries(high, "high")
	if err != nil {
		return nil, err
	}
	lo, err := positiveSeries(low, "low")
	if err != nil {
		return nil, err
	}
	cl, err := positiveSeries(close, "close")
	if err != nil {
		return nil, err
	}
	n := len(cl)
	if len(op) != n || len(hi) != n || len(lo) != n {
		return nil, fmt.Errorf("open, high, low and close must have the same length, got %d, %d, %d and %d", len(op), len(hi), len(lo), n)
	}
	for i := range cl {
		if hi[i] < lo[i] {
			return nil, fmt.Errorf("high %v is below low %v at row index %d", hi[i], lo[i], i)
		}
	}
	var vol []float64
	if !insyra.IsNilList(volume) {
		if vol, err = numericSeries(volume, "volume"); err != nil {
			return nil, err
		}
		if len(vol) != n {
			return nil, fmt.Errorf("volume has %d rows but close has %d", len(vol), n)
		}
		for i, v := range vol {
			if v < 0 {
				return nil, fmt.Errorf("volume holds %v at row index %d; it must not be negative", v, i)
			}
		}
	}
	starts, err := cleanSegmentStarts(o.SegmentStarts, n)
	if err != nil {
		return nil, err
	}

	outO, outH, outL, outC := make([]any, n), make([]any, n), make([]any, n), make([]any, n)
	outV := make([]any, n)
	rng := newBootstrapRNG(o.Seed)
	for _, b := range segmentBounds(n, starts) {
		from, to := b[0], b[1]
		if to == from {
			continue
		}
		outO[from], outH[from], outL[from], outC[from] = op[from], hi[from], lo[from], cl[from]
		if vol != nil {
			outV[from] = vol[from]
		}
		order := make([]int, 0, to-from-1)
		for i := from + 1; i < to; i++ {
			order = append(order, i)
		}
		for i := len(order) - 1; i > 0; i-- {
			j := int(rng.uintN(uint64(i + 1)))
			order[i], order[j] = order[j], order[i]
		}
		prev := cl[from]
		for k, src := range order {
			row := from + 1 + k
			barOpen := prev * (op[src] / cl[src-1])
			outO[row] = barOpen
			outH[row] = barOpen * (hi[src] / op[src])
			outL[row] = barOpen * (lo[src] / op[src])
			prev = barOpen * (cl[src] / op[src])
			outC[row] = prev
			if vol != nil {
				outV[row] = vol[src]
			}
		}
	}
	cols := []*insyra.DataList{
		insyra.NewDataList(outO...).SetName("Open"),
		insyra.NewDataList(outH...).SetName("High"),
		insyra.NewDataList(outL...).SetName("Low"),
		insyra.NewDataList(outC...).SetName("Close"),
	}
	if vol != nil {
		cols = append(cols, insyra.NewDataList(outV...).SetName("Volume"))
	}
	return insyra.NewDataTable(cols...), nil
}
