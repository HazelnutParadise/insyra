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
// volume is NaN. The first n-1 rows are NaN.
func VolumeProfile(high, low, close, volume insyra.IDataList, n int, opts ...VolumeProfileOptions) (*VolumeProfileResult, error) {
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
	h, l, _, v, err := barsWithVolume(high, low, close, volume)
	if err != nil {
		return nil, err
	}
	poc, vah, val := nanSeries(len(h)), nanSeries(len(h)), nanSeries(len(h))
	bins := make([]float64, o.Bins)
	for i := n - 1; i < len(h); i++ {
		lo, hi := l[i], h[i]
		for j := i - n + 1; j < i; j++ {
			lo, hi = math.Min(lo, l[j]), math.Max(hi, h[j])
		}
		if hi == lo {
			continue
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
			continue
		}
		best := 0
		for k := range bins {
			if bins[k] > bins[best] {
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
			if above > below {
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
	return &VolumeProfileResult{listOfFloats("POC", poc), listOfFloats("ValueAreaHigh", vah), listOfFloats("ValueAreaLow", val)}, nil
}
