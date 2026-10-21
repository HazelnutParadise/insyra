package ta

import (
	"errors"
	"fmt"
	"math"

	"github.com/HazelnutParadise/insyra"
)

// VolumeProfileOptions are the optional settings of VolumeProfile. A zero field
// takes the default shown.
type VolumeProfileOptions struct {
	// Bins is how many equal price bins the window's range is cut into. Default 24.
	Bins int
	// ValueArea is the share of the window's volume the value area holds, from 0
	// to 1. Default 0.7.
	ValueArea float64
	// SegmentStarts are the rows on which a new data segment starts, for
	// example the first row after a trading halt or a relisting. Each segment is
	// computed on its own: the window restarts at a start, so it never holds
	// rows from before it, and a segment shows NaN until its window is full. Row
	// 0 starts the first segment and need not be listed. Left empty, the result
	// does not change.
	SegmentStarts []int
}

// VolumeProfileResult holds, for every row, the price levels of the volume
// profile of the window ending on it.
type VolumeProfileResult struct {
	POC           *insyra.DataList // point of control: the middle of the price bin with the most volume
	ValueAreaHigh *insyra.DataList // top of the value area
	ValueAreaLow  *insyra.DataList // bottom of the value area
}

// VolumeProfile returns, for every row, the volume profile of the last n rows
// (the current one included): the window's price range from its lowest low to
// its highest high is cut into Bins equal bins, and each bar's volume is spread
// over the bins its range covers in proportion to the overlap. The point of
// control is the middle of the bin with the most volume (the lowest one on a
// tie). The value area is the run of bins around it that holds ValueArea of the
// volume, grown one bin at a time toward the side with more volume; its high
// and low are the outer edges of that run. A window with no price range or no
// volume is NaN. The first n-1 rows are NaN, and so are the first n-1 rows of
// every segment in VolumeProfileOptions.SegmentStarts.
func VolumeProfile(high, low, volume insyra.IDataList, n int, opts ...VolumeProfileOptions) (*VolumeProfileResult, error) {
	if err := checkPeriod("n", n, 1); err != nil {
		return nil, err
	}
	if len(opts) > 1 {
		return nil, errors.New("VolumeProfile accepts at most one VolumeProfileOptions")
	}
	var o VolumeProfileOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	if o.Bins < 0 {
		return nil, errors.New("option Bins must not be negative")
	}
	if o.Bins == 0 {
		o.Bins = 24
	}
	if math.IsNaN(o.ValueArea) || o.ValueArea < 0 || o.ValueArea > 1 {
		return nil, fmt.Errorf("option ValueArea must be between 0 and 1, got %v", o.ValueArea)
	}
	if o.ValueArea == 0 {
		o.ValueArea = 0.7
	}
	h, l, err := highLowSeries(high, low)
	if err != nil {
		return nil, err
	}
	v, err := volumeSeries(volume)
	if err != nil {
		return nil, err
	}
	if len(v) != len(h) {
		return nil, fmt.Errorf("volume has %d rows but high has %d", len(v), len(h))
	}
	starts, err := cleanSegmentStarts(o.SegmentStarts, len(h))
	if err != nil {
		return nil, err
	}
	poc, vah, val := nanSeries(len(h)), nanSeries(len(h)), nanSeries(len(h))
	bins := make([]float64, o.Bins)
	for _, seg := range segmentBounds(len(h), starts) {
		for i := seg[0] + n - 1; i < seg[1]; i++ {
			volumeProfileWindow(h, l, v, i, n, o, bins, poc, vah, val)
		}
	}
	return &VolumeProfileResult{listOfFloats("POC", poc), listOfFloats("ValueAreaHigh", vah), listOfFloats("ValueAreaLow", val)}, nil
}

// tieTolerance is the share of a window's volume by which two bins may differ
// and still count as tied. The overlaps that split a bar's volume are computed
// from rounded price edges, so bins that hold equal volume differ by about 1e-16
// of it; without the tolerance a spread bar would not tie the way it should.
const tieTolerance = 1e-9

// volumeProfileWindow fills row i of poc, vah and val from the n rows ending
// on it, leaving them NaN when the window has no price range or no volume.
func volumeProfileWindow(h, l, v []float64, i, n int, o VolumeProfileOptions, bins, poc, vah, val []float64) {
	lo, hi := l[i], h[i]
	for j := i - n + 1; j < i; j++ {
		lo, hi = math.Min(lo, l[j]), math.Max(hi, h[j])
	}
	if hi == lo {
		return
	}
	for k := range bins {
		bins[k] = 0
	}
	width := (hi - lo) / float64(o.Bins)
	var total float64
	for j := i - n + 1; j <= i; j++ {
		total += v[j]
		if h[j] == l[j] {
			// a bar with no range puts all its volume in the bin it sits in
			k := min(int((h[j]-lo)/width), o.Bins-1)
			bins[k] += v[j]
			continue
		}
		first := min(int((l[j]-lo)/width), o.Bins-1)
		last := min(int((h[j]-lo)/width), o.Bins-1)
		span := h[j] - l[j]
		for k := first; k <= last; k++ {
			overlap := math.Min(h[j], lo+float64(k+1)*width) - math.Max(l[j], lo+float64(k)*width)
			if overlap > 0 {
				bins[k] += v[j] * overlap / span
			}
		}
	}
	if total == 0 {
		return
	}
	best := 0
	for k := range bins {
		if bins[k] > bins[best]+tieTolerance*total {
			best = k
		}
	}
	a, b := best, best
	held := bins[best]
	for held < o.ValueArea*total && (a > 0 || b < o.Bins-1) {
		var below, above float64 = -1, -1
		if a > 0 {
			below = bins[a-1]
		}
		if b < o.Bins-1 {
			above = bins[b+1]
		}
		if above > below+tieTolerance*total {
			b++
			held += bins[b]
		} else {
			a--
			held += bins[a]
		}
	}
	poc[i] = lo + (float64(best)+0.5)*width
	val[i], vah[i] = lo+float64(a)*width, lo+float64(b+1)*width
}
