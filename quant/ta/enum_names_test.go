package ta

import (
	"fmt"
	"testing"
)

// TestEnumsPrintTheirNames pins enum-conventions: every numeric enum
// prints a name, and a value outside its constants prints TypeName(n).
func TestEnumsPrintTheirNames(t *testing.T) {
	cases := []struct {
		got  fmt.Stringer
		want string
	}{
		{RSIWilder, "RSIWilder"},
		{RSISimple, "RSISimple"},
		{RSIMethod(99), "RSIMethod(99)"},
		{RSIMissingError, "RSIMissingError"},
		{RSIMissingFlat, "RSIMissingFlat"},
		{RSIMissingBreak, "RSIMissingBreak"},
		{RSIMissing(99), "RSIMissing(99)"},
		{KDRecursive, "KDRecursive"},
		{KDMovingAverage, "KDMovingAverage"},
		{KDMethod(99), "KDMethod(99)"},
		{CRMidHighLow, "CRMidHighLow"},
		{CRMidWeightedClose, "CRMidWeightedClose"},
		{CRMidTypical, "CRMidTypical"},
		{CRMidOHLC, "CRMidOHLC"},
		{CRMidpoint(99), "CRMidpoint(99)"},
		{PatternHeadAndShouldersTop, "PatternHeadAndShouldersTop"},
		{PatternHeadAndShouldersBottom, "PatternHeadAndShouldersBottom"},
		{PatternDoubleTop, "PatternDoubleTop"},
		{PatternDoubleBottom, "PatternDoubleBottom"},
		{PatternRectangle, "PatternRectangle"},
		{PatternAscendingTriangle, "PatternAscendingTriangle"},
		{PatternDescendingTriangle, "PatternDescendingTriangle"},
		{PatternSymmetricalTriangle, "PatternSymmetricalTriangle"},
		{PatternRisingWedge, "PatternRisingWedge"},
		{PatternFallingWedge, "PatternFallingWedge"},
		{PatternDiamond, "PatternDiamond"},
		{PatternKind(99), "PatternKind(99)"},
		{TrendUnknown, "TrendUnknown"},
		{TrendUp, "TrendUp"},
		{TrendDown, "TrendDown"},
		{Trend(99), "Trend(99)"},
		{BreakNone, "BreakNone"},
		{BreakUp, "BreakUp"},
		{BreakDown, "BreakDown"},
		{BreakDirection(99), "BreakDirection(99)"},
		{GapCommon, "GapCommon"},
		{GapBreakaway, "GapBreakaway"},
		{GapRunaway, "GapRunaway"},
		{GapExhaustion, "GapExhaustion"},
		{GapUnclassified, "GapUnclassified"},
		{GapKind(99), "GapKind(99)"},
		{GapFillUnknown, "GapFillUnknown"},
		{GapFilled, "GapFilled"},
		{GapNotFilled, "GapNotFilled"},
		{GapFill(99), "GapFill(99)"},
	}
	for _, c := range cases {
		if got := fmt.Sprint(c.got); got != c.want {
			t.Errorf("%T: got %q, want %q", c.got, got, c.want)
		}
	}
}
