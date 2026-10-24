package ta

import (
	"errors"
	"math"

	"github.com/HazelnutParadise/insyra"
)

// FilterRuleOptions are the optional settings of FilterRule.
type FilterRuleOptions struct {
	// LongOnly makes a sell signal go flat (0) instead of short (-1).
	LongOnly bool
	// SegmentStarts are the rows on which a new data segment starts, as in
	// ZigZagOptions. The position is flat on the first row of a segment and the
	// rule starts again from there: nothing is carried over a break.
	SegmentStarts []int
}

// FilterRule is Alexander's filter rule (1961), the trading rule Fama and Blume
// (1966) used to test whether price trends can be traded: buy when the price has
// risen threshold from its lowest close since the last sell, sell (and, unless
// LongOnly, sell short) when it has fallen threshold from its highest close
// since the last buy. threshold is a fraction such as 0.05 for a 5% filter.
//
// It returns one position per row: 1 long, 0 flat, -1 short. The position on a
// row is decided from the closes up to and including that row, so it is the
// position to hold from the next row; BacktestSignal in the quant package
// applies it one row late. It is flat until the first threshold move has
// happened. The turning points are those of ZigZag, so a move of exactly
// threshold counts and the rule uses no later price.
//
// BacktestSignal accepts the long-short result as it is; with LongOnly the
// positions stay between 0 and 1.
func FilterRule(prices insyra.IDataList, threshold float64, opts ...FilterRuleOptions) (*insyra.DataList, error) {
	if math.IsNaN(threshold) || threshold <= 0 || threshold >= 1 {
		return nil, errors.New("threshold must be a fraction between 0 and 1, such as 0.05 for a 5% filter")
	}
	if len(opts) > 1 {
		return nil, errors.New("at most one FilterRuleOptions is accepted")
	}
	var o FilterRuleOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	p, err := positivePrices(prices)
	if err != nil {
		return nil, err
	}
	starts, err := cleanSegmentStarts(o.SegmentStarts, len(p))
	if err != nil {
		return nil, err
	}
	th := make([]float64, len(p))
	for i := range th {
		th[i] = threshold
	}
	swings := zigzagCore(p, th, starts)

	sell := -1.0
	if o.LongOnly {
		sell = 0
	}
	pos := make([]float64, len(p))
	bounds := segmentBounds(len(p), starts)
	for i, s := range swings {
		end := bounds[s.Segment][1]
		if i+1 < len(swings) && swings[i+1].Segment == s.Segment {
			end = swings[i+1].ConfirmedAt
		}
		held := 1.0
		if s.Kind == SwingHigh {
			held = sell
		}
		for t := s.ConfirmedAt; t < end; t++ {
			pos[t] = held
		}
	}
	return listOfFloats("filter_rule", pos), nil
}
