package ta

import (
	"errors"
	"fmt"
	"math"
	"sort"

	"github.com/HazelnutParadise/insyra"
)

// PatternKind names a chart pattern found by DetectPatterns.
type PatternKind int

const (
	// PatternHeadAndShouldersTop is three peaks with the middle one highest,
	// broken downward through the line joining the two troughs (the neckline).
	PatternHeadAndShouldersTop PatternKind = iota
	// PatternHeadAndShouldersBottom is its mirror: three troughs with the middle
	// one lowest, broken upward through the neckline.
	PatternHeadAndShouldersBottom
	// PatternDoubleTop is two peaks of about the same height, broken downward
	// through the trough between them.
	PatternDoubleTop
	// PatternDoubleBottom is two troughs of about the same depth, broken upward
	// through the peak between them.
	PatternDoubleBottom
	// PatternRectangle is a flat top and a flat bottom, broken in either direction.
	PatternRectangle
	// PatternAscendingTriangle is a flat top over a rising bottom.
	PatternAscendingTriangle
	// PatternDescendingTriangle is a falling top over a flat bottom.
	PatternDescendingTriangle
	// PatternSymmetricalTriangle is a falling top over a rising bottom.
	PatternSymmetricalTriangle
	// PatternRisingWedge is a top and a bottom that both rise and close in
	// on each other.
	PatternRisingWedge
	// PatternFallingWedge is a top and a bottom that both fall and close in
	// on each other.
	PatternFallingWedge
	// PatternDiamond is a widening followed by a narrowing, broken downward.
	PatternDiamond
)

// String returns the constant name of k, or PatternKind(n) for a value that is
// not one of the constants.
func (k PatternKind) String() string {
	switch k {
	case PatternHeadAndShouldersTop:
		return "PatternHeadAndShouldersTop"
	case PatternHeadAndShouldersBottom:
		return "PatternHeadAndShouldersBottom"
	case PatternDoubleTop:
		return "PatternDoubleTop"
	case PatternDoubleBottom:
		return "PatternDoubleBottom"
	case PatternRectangle:
		return "PatternRectangle"
	case PatternAscendingTriangle:
		return "PatternAscendingTriangle"
	case PatternDescendingTriangle:
		return "PatternDescendingTriangle"
	case PatternSymmetricalTriangle:
		return "PatternSymmetricalTriangle"
	case PatternRisingWedge:
		return "PatternRisingWedge"
	case PatternFallingWedge:
		return "PatternFallingWedge"
	case PatternDiamond:
		return "PatternDiamond"
	}
	return fmt.Sprintf("PatternKind(%d)", int(k))
}

// Trend is the direction of the price change before a pattern.
type Trend int

const (
	// TrendUnknown means there were not enough rows in the segment to tell.
	TrendUnknown Trend = iota
	// TrendUp means the close was higher than TrendRows rows earlier.
	TrendUp
	// TrendDown means it was not higher.
	TrendDown
)

// String returns the constant name of t, or Trend(n) for a value that is not one
// of the constants.
func (t Trend) String() string {
	switch t {
	case TrendUnknown:
		return "TrendUnknown"
	case TrendUp:
		return "TrendUp"
	case TrendDown:
		return "TrendDown"
	}
	return fmt.Sprintf("Trend(%d)", int(t))
}

// BreakDirection says which way a price left a pattern or crossed a line.
type BreakDirection int

const (
	// BreakNone means no break.
	BreakNone BreakDirection = iota
	// BreakUp is a close above the line.
	BreakUp
	// BreakDown is a close below the line.
	BreakDown
)

// String returns the constant name of d, or BreakDirection(n) for a value that
// is not one of the constants.
func (d BreakDirection) String() string {
	switch d {
	case BreakNone:
		return "BreakNone"
	case BreakUp:
		return "BreakUp"
	case BreakDown:
		return "BreakDown"
	}
	return fmt.Sprintf("BreakDirection(%d)", int(d))
}

// Pattern is one chart pattern candidate. Every row in it is a row index.
//
// ConfirmedAt is the row that confirmed the last turning point of the pattern,
// the first day it could be known; Start is the row of its first turning
// point, which is in the past by then. The breakout is the first close beyond
// the pattern's line before ExpiresAt. BreakoutAt is -1 and Direction is
// BreakNone for a pattern that expired, or that the price invalidated by
// passing the head or the top of the double, before it broke out. A pattern
// that breaks the way its kind does not allow (a head and shoulders top
// breaking up) never breaks: only the textbook directions are watched, a box
// is watched both ways.
//
// PriorTrend is the direction of the price change over the TrendRows before Start,
// +1 for up, -1 for down and 0 when there were not that many rows in the
// segment. Candidates overlap: each new turning point can start a new one, so
// several can break out on the same day.
type Pattern struct {
	Kind        PatternKind
	Start       int
	ConfirmedAt int
	ExpiresAt   int
	BreakoutAt  int
	Direction   BreakDirection
	PriorTrend  Trend
}

// PatternOptions are the optional settings of DetectPatterns. A zero field
// takes the default shown.
type PatternOptions struct {
	// SegmentStarts are the rows on which a new data segment starts. Pass the SegmentStarts
	// given to ZigZagAdaptive. No pattern or line crosses a break.
	SegmentStarts []int
	// FlatTolerance is how far a pattern's top or bottom may move, as a share
	// of the pattern's height from highest peak to lowest trough, and still
	// count as flat. Default 0.2.
	FlatTolerance float64
	// SymmetryTolerance is how far apart, as a share of the pattern's height,
	// the two shoulders of a head and shoulders (and the two necks) or the two
	// peaks of a double top may be. Default 0.3.
	SymmetryTolerance float64
	// MaxRows caps how long a pattern may wait for its breakout after its last
	// turning point: the width of a box or triangle, or the span from the first
	// peak to the last turning point of the others. Default 120.
	MaxRows int
	// MinDoubleSeparation is the fewest rows between the two peaks of a double
	// top or bottom. Default 5.
	MinDoubleSeparation int
	// TrendRows is how many rows before a pattern's start PriorTrend looks back.
	// Default 60.
	TrendRows int
}

type patternLine struct{ x1, y1, x2, y2 float64 }

func (l patternLine) at(t int) float64 {
	return l.y1 + (l.y2-l.y1)*(float64(t)-l.x1)/(l.x2-l.x1)
}

type patternCandidate struct {
	Pattern
	upper, lower patternLine
	mode         int // -1 watch the lower line only, +1 the upper line only, 0 both
	invHigh      float64
	invLow       float64
}

func checkPivots(pivots []Pivot, n int) error {
	for i, p := range pivots {
		if p.Index < 0 || p.ConfirmedAt <= p.Index || p.ConfirmedAt >= n {
			return fmt.Errorf("pivot at list position %d has Index %d and ConfirmedAt %d, which do not fit %d rows", i, p.Index, p.ConfirmedAt, n)
		}
		if i > 0 && (p.ConfirmedAt < pivots[i-1].ConfirmedAt || p.Segment < pivots[i-1].Segment || p.Index <= pivots[i-1].Index) {
			return fmt.Errorf("pivot at list position %d is not after the one before it; pass the pivots as ZigZag returns them", i)
		}
	}
	return nil
}

// checkSegments refuses pivots whose Segment does not match the segment their
// row lies in under segment starts, which is what happens when the pivots were made
// with different SegmentStarts than the ones passed now.
func checkSegments(pivots []Pivot, starts []int) error {
	for i, p := range pivots {
		seg := sort.SearchInts(starts, p.Index+1) // starts at or before the pivot's row
		if p.Segment != seg {
			return fmt.Errorf("pivot at list position %d has Segment %d but row index %d lies in segment %d of the SegmentStarts given; pass the SegmentStarts used for ZigZagAdaptive", i, p.Segment, p.Index, seg)
		}
	}
	return nil
}

// segmentStarts returns, for every row, the first row of its segment.
func segmentStartOf(n int, starts []int) []int {
	start := make([]int, n)
	for _, b := range segmentBounds(n, starts) {
		for i := b[0]; i < b[1]; i++ {
			start[i] = b[0]
		}
	}
	return start
}

// DetectPatterns finds the chart patterns formed by the turning points of
// ZigZag or ZigZagAdaptive, using only information available at the close of each
// row: a pattern is first known on the row that confirms its last turning
// point, and its breakout is the first later close beyond its line. The
// pivots must come from the same prices.
//
// It watches head and shoulders tops and bottoms, double tops and bottoms,
// boxes, ascending, descending and symmetric triangles, rising and falling
// wedges, and diamonds. How flat a line must be to count as flat and how
// alike two peaks must be are set in PatternOptions.
//
// Prices are closes. Both a pattern's lines and its breakout are measured on
// closes, as the turning points are, so they are not the lines a chart
// reader would draw through intraday highs and lows. The same pattern is
// found again from every turning point that completes it, so a long
// sideways stretch yields many overlapping boxes.
func DetectPatterns(prices insyra.IDataList, pivots []Pivot, opts ...PatternOptions) ([]Pattern, error) {
	if len(opts) > 1 {
		return nil, errors.New("DetectPatterns accepts at most one PatternOptions")
	}
	var o PatternOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	if o.FlatTolerance < 0 || o.SymmetryTolerance < 0 || o.MaxRows < 0 || o.MinDoubleSeparation < 0 || o.TrendRows < 0 {
		return nil, errors.New("PatternOptions fields must not be negative")
	}
	if o.FlatTolerance == 0 {
		o.FlatTolerance = 0.2
	}
	if o.SymmetryTolerance == 0 {
		o.SymmetryTolerance = 0.3
	}
	if o.MaxRows == 0 {
		o.MaxRows = 120
	}
	if o.MinDoubleSeparation == 0 {
		o.MinDoubleSeparation = 5
	}
	if o.TrendRows == 0 {
		o.TrendRows = 60
	}
	c, err := positivePrices(prices)
	if err != nil {
		return nil, err
	}
	if err := checkPivots(pivots, len(c)); err != nil {
		return nil, err
	}
	starts, err := cleanSegmentStarts(o.SegmentStarts, len(c))
	if err != nil {
		return nil, err
	}
	if err := checkSegments(pivots, starts); err != nil {
		return nil, err
	}
	segStart := segmentStartOf(len(c), starts)

	var cands []*patternCandidate
	var live []*patternCandidate
	next := 0 // pivots confirmed so far
	for t := range c {
		if segStart[t] == t {
			live = live[:0]
		}
		for next < len(pivots) && pivots[next].ConfirmedAt <= t {
			p := next
			next++
			for _, cand := range newCandidates(pivots, p, t, c, segStart, o) {
				cands = append(cands, cand)
				live = append(live, cand)
			}
		}
		if segStart[t] == t {
			continue
		}
		kept := live[:0]
		for _, cand := range live {
			if t > cand.ExpiresAt || c[t] > cand.invHigh || c[t] < cand.invLow {
				continue
			}
			d := BreakNone
			if cand.mode >= 0 && c[t] > cand.upper.at(t) {
				d = BreakUp
			} else if cand.mode <= 0 && c[t] < cand.lower.at(t) {
				d = BreakDown
			}
			if d == BreakNone {
				kept = append(kept, cand)
				continue
			}
			cand.BreakoutAt = t
			cand.Direction = d
		}
		live = kept
	}
	out := make([]Pattern, len(cands))
	for i, cand := range cands {
		out[i] = cand.Pattern
	}
	return out, nil
}

// newCandidates returns the patterns whose last turning point is pivots[p],
// confirmed on row t.
func newCandidates(pivots []Pivot, p, t int, c []float64, segStart []int, o PatternOptions) []*patternCandidate {
	var out []*patternCandidate
	price := func(i int) float64 { return pivots[i].Price }
	row := func(i int) int { return pivots[i].Index }
	sameSegment := func(back int) bool {
		return p >= back && pivots[p].Segment == pivots[p-back].Segment
	}
	line := func(a, b int) patternLine {
		return patternLine{float64(row(a)), price(a), float64(row(b)), price(b)}
	}
	priorTrend := func(x0 int) Trend {
		if x0-o.TrendRows < 0 || x0-segStart[x0] < o.TrendRows {
			return TrendUnknown
		}
		if c[x0] > c[x0-o.TrendRows] {
			return TrendUp
		}
		return TrendDown
	}
	base := func(kind PatternKind, first int, expires int) patternCandidate {
		return patternCandidate{
			Pattern: Pattern{Kind: kind, Start: row(first), ConfirmedAt: t, ExpiresAt: expires, BreakoutAt: -1},
			invHigh: math.Inf(1),
			invLow:  math.Inf(-1),
		}
	}

	// Two peaks and two troughs: box, triangles, wedges.
	if sameSegment(3) {
		var hA, hB, lA, lB int
		if pivots[p].Kind == PivotHigh {
			hA, hB, lA, lB = p-2, p, p-3, p-1
		} else {
			hA, hB, lA, lB = p-3, p-1, p-2, p
		}
		height := math.Max(price(hA), price(hB)) - math.Min(price(lA), price(lB))
		if height > 0 {
			du := (price(hB) - price(hA)) / height
			dl := (price(lB) - price(lA)) / height
			f := o.FlatTolerance
			kind, ok := PatternRectangle, true
			switch {
			case math.Abs(du) <= f && math.Abs(dl) <= f:
				kind = PatternRectangle
			case math.Abs(du) <= f && dl > f:
				kind = PatternAscendingTriangle
			case du < -f && math.Abs(dl) <= f:
				kind = PatternDescendingTriangle
			case du < -f && dl > f:
				kind = PatternSymmetricalTriangle
			case du > f && dl > f && dl > du:
				kind = PatternRisingWedge
			case du < -f && dl < -f && du < dl:
				kind = PatternFallingWedge
			default:
				ok = false
			}
			if ok {
				width := min(row(p)-row(p-3), o.MaxRows)
				cand := base(kind, p-3, row(p)+width)
				cand.upper, cand.lower = line(hA, hB), line(lA, lB)
				cand.PriorTrend = priorTrend(row(p - 3))
				out = append(out, &cand)
			}
		}
	}

	for _, sgn := range []int{1, -1} {
		s := float64(sgn)
		want := PivotHigh
		if sgn == -1 {
			want = PivotLow
		}
		if pivots[p].Kind != want {
			continue
		}
		// Head and shoulders: the last turning point is the right shoulder.
		if sameSegment(5) {
			l0, h1, l1, h2, l2, h3 := p-5, p-4, p-3, p-2, p-1, p
			head := s * price(h2)
			if head > s*price(h1) && head > s*price(h3) && s*price(l0) < math.Min(s*price(l1), s*price(l2)) {
				height := s * (price(h2) - (price(l1)+price(l2))/2)
				if height > 0 && math.Abs(price(h1)-price(h3)) <= o.SymmetryTolerance*height && math.Abs(price(l1)-price(l2)) <= o.SymmetryTolerance*height {
					kind := PatternHeadAndShouldersTop
					if sgn == -1 {
						kind = PatternHeadAndShouldersBottom
					}
					cand := base(kind, h1, row(p)+min(row(p)-row(h1), o.MaxRows))
					cand.upper, cand.lower = line(l1, l2), line(l1, l2)
					cand.mode = -sgn
					if sgn == 1 {
						cand.invHigh = price(h2)
					} else {
						cand.invLow = price(h2)
					}
					out = append(out, &cand)
				}
			}
		}
		// Double top or bottom: the last turning point is the second peak.
		if sameSegment(3) {
			l0, h1, l1, h2 := p-3, p-2, p-1, p
			top := math.Max(s*price(h1), s*price(h2))
			height := top - s*price(l1)
			if height > 0 && math.Abs(price(h1)-price(h2)) <= o.SymmetryTolerance*height &&
				row(h2)-row(h1) >= o.MinDoubleSeparation && s*price(l0) < s*price(l1) {
				kind := PatternDoubleTop
				if sgn == -1 {
					kind = PatternDoubleBottom
				}
				cand := base(kind, h1, row(p)+min(row(p)-row(h1), o.MaxRows))
				flat := patternLine{float64(row(l1)), price(l1), float64(row(l1) + 1), price(l1)}
				cand.upper, cand.lower = flat, flat
				cand.mode = -sgn
				if sgn == 1 {
					cand.invHigh = s * top
				} else {
					cand.invLow = s * top
				}
				out = append(out, &cand)
			}
		}
	}

	// Diamond: the last turning point is the third trough.
	if pivots[p].Kind == PivotLow && sameSegment(6) {
		l0, h1, l1, h2, l2, h3, l3 := p-6, p-5, p-4, p-3, p-2, p-1, p
		if price(h2) > price(h1) && price(l2) < price(l1) && price(h3) < price(h2) && price(l3) > price(l2) && price(l0) < price(l2) {
			cand := base(PatternDiamond, h1, row(p)+min(row(p)-row(h1), o.MaxRows))
			cand.upper, cand.lower = line(l2, l3), line(l2, l3)
			cand.mode = -1
			cand.invHigh = price(h2)
			out = append(out, &cand)
		}
	}
	return out
}

// FanBreak is one close through a fan line, as found by FanBreaks.
//
// Direction is BreakDown for a fan drawn under rising troughs and BreakUp for
// a fan drawn over falling peaks. Line is 1, 2 or 3, the count of fan lines
// the price has now broken; after the third the fan is spent. Anchor and
// Through are the rows of the two turning points the broken line joins.
type FanBreak struct {
	Direction BreakDirection
	Line      int
	Row       int
	Anchor    int
	Through   int
}

// FanOptions are the optional settings of FanBreaks.
type FanOptions struct {
	// SegmentStarts are the rows on which a new data segment starts. Pass the SegmentStarts
	// given to ZigZagAdaptive.
	SegmentStarts []int
}

// FanBreaks finds fan-principle breaks in the turning points of ZigZag or
// ZigZagAdaptive, using only information known at each row. From a trough (or peak)
// it draws a line through the next higher trough (lower peak) that is flatter
// than the last line; each close through such a line is a break, and the third
// ends the fan. A close past the anchor itself discards the fan. Pass the
// SegmentStarts given to ZigZagAdaptive in FanOptions.
func FanBreaks(prices insyra.IDataList, pivots []Pivot, opts ...FanOptions) ([]FanBreak, error) {
	if len(opts) > 1 {
		return nil, errors.New("FanBreaks accepts at most one FanOptions")
	}
	var o FanOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	c, err := positivePrices(prices)
	if err != nil {
		return nil, err
	}
	if err := checkPivots(pivots, len(c)); err != nil {
		return nil, err
	}
	starts, err := cleanSegmentStarts(o.SegmentStarts, len(c))
	if err != nil {
		return nil, err
	}
	if err := checkSegments(pivots, starts); err != nil {
		return nil, err
	}
	segStart := segmentStartOf(len(c), starts)

	type fan struct {
		anchor, through, broken int
		slope                   float64
	}
	var fans [2]fan // [0] fan over rising troughs, [1] fan over falling peaks
	reset := func(f *fan) { *f = fan{anchor: -1, through: -1, slope: math.Inf(1)} }
	reset(&fans[0])
	reset(&fans[1])

	var out []FanBreak
	next := 0
	for t := range c {
		if segStart[t] == t {
			reset(&fans[0])
			reset(&fans[1])
		}
		for next < len(pivots) && pivots[next].ConfirmedAt <= t {
			p := next
			next++
			for side := 0; side < 2; side++ {
				want := PivotLow
				if side == 1 {
					want = PivotHigh
				}
				if pivots[p].Kind != want {
					continue
				}
				f := &fans[side]
				if f.anchor < 0 || pivots[f.anchor].Segment != pivots[p].Segment {
					reset(f)
					f.anchor = p
					continue
				}
				a := f.anchor
				beyond := pivots[p].Price <= pivots[a].Price
				if side == 1 {
					beyond = pivots[p].Price >= pivots[a].Price
				}
				if beyond {
					reset(f)
					f.anchor = p
					continue
				}
				if f.through < 0 {
					slope := math.Abs(pivots[p].Price-pivots[a].Price) / float64(pivots[p].Index-pivots[a].Index)
					if slope < f.slope {
						f.through = p
					}
				}
			}
		}
		if segStart[t] == t {
			continue
		}
		for side := 0; side < 2; side++ {
			f := &fans[side]
			if f.anchor < 0 {
				continue
			}
			a := pivots[f.anchor]
			if (side == 0 && c[t] < a.Price) || (side == 1 && c[t] > a.Price) {
				reset(f)
				continue
			}
			if f.through < 0 || pivots[f.through].ConfirmedAt > t {
				continue
			}
			b := pivots[f.through]
			level := patternLine{float64(a.Index), a.Price, float64(b.Index), b.Price}.at(t)
			crossed := c[t] < level
			dir := BreakDown
			if side == 1 {
				crossed = c[t] > level
				dir = BreakUp
			}
			if !crossed {
				continue
			}
			f.broken++
			f.slope = math.Abs(b.Price-a.Price) / float64(b.Index-a.Index)
			out = append(out, FanBreak{Direction: dir, Line: min(f.broken, 3), Row: t, Anchor: a.Index, Through: b.Index})
			f.through = -1
			if f.broken >= 3 {
				f.anchor, f.broken, f.slope = -1, 0, math.Inf(1)
			}
		}
	}
	return out, nil
}
