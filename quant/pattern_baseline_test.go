package quant

import (
	"encoding/json"
	"os"
	"sync"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// baselineCase is one synthetic series with the values an independent
// implementation produced for it; testdata/pattern_baseline.py regenerates them.
type baselineCase struct {
	Name     string
	Multiple float64
	Breaks   []int
	Open     []float64
	High     []float64
	Low      []float64
	Close    []float64
	Volume   []float64
	Pivots   []struct {
		Kind, Index, Confirmed, Segment int
		Price                           float64
	}
	Patterns []struct{ Kind, Formed, Breakout, Direction, Pretrend int }
	Fans     []struct{ Direction, Number, At int }
	Gaps     []struct {
		Row            int
		Up             bool
		Label          string
		RelativeVolume float64
		TrendZ         float64
		Fill5, Fill20  *bool
	}
}

var (
	baselineOnce  sync.Once
	baselineCases []baselineCase
)

func baseline(t *testing.T) []baselineCase {
	t.Helper()
	baselineOnce.Do(func() {
		raw, err := os.ReadFile("testdata/pattern_baseline.json")
		if err != nil {
			t.Fatal(err)
		}
		var f struct{ Cases []baselineCase }
		if err := json.Unmarshal(raw, &f); err != nil {
			t.Fatal(err)
		}
		baselineCases = f.Cases
	})
	return baselineCases
}

func listOf(v []float64) *insyra.DataList { return insyra.NewDataList(toAny(v)...) }

func baselinePivots(t *testing.T, c baselineCase) []Pivot {
	t.Helper()
	th, err := VolatilityThresholds(listOf(c.Close), VolatilityThresholdOptions{Multiple: c.Multiple, SegmentStarts: c.Breaks})
	if err != nil {
		t.Fatal(err)
	}
	pivots, err := ZigZagAdaptive(listOf(c.Close), th, ZigZagOptions{SegmentStarts: c.Breaks})
	if err != nil {
		t.Fatal(err)
	}
	return pivots
}

func TestZigZagByMatchesTheReference(t *testing.T) {
	for _, c := range baseline(t) {
		got := baselinePivots(t, c)
		if len(got) != len(c.Pivots) {
			t.Fatalf("%s: %d pivots, reference has %d", c.Name, len(got), len(c.Pivots))
		}
		for i, w := range c.Pivots {
			kind := PivotLow
			if w.Kind == 1 {
				kind = PivotHigh
			}
			want := Pivot{kind, w.Index, w.Price, w.Confirmed, w.Segment}
			if got[i] != want {
				t.Fatalf("%s pivot %d: got %+v, reference %+v", c.Name, i, got[i], want)
			}
		}
	}
}

func TestDetectPatternsMatchesTheReference(t *testing.T) {
	kindsSeen := map[PatternKind]bool{}
	for _, c := range baseline(t) {
		got, err := DetectPatterns(listOf(c.Close), baselinePivots(t, c), PatternOptions{SegmentStarts: c.Breaks})
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != len(c.Patterns) {
			t.Fatalf("%s: %d patterns, reference has %d", c.Name, len(got), len(c.Patterns))
		}
		for i, w := range c.Patterns {
			dir := BreakNone
			switch w.Direction {
			case 1:
				dir = BreakUp
			case -1:
				dir = BreakDown
			}
			g := got[i]
			if g.Kind != PatternKind(w.Kind) || g.ConfirmedAt != w.Formed || g.BreakoutAt != w.Breakout || g.Direction != dir || g.PriorTrend != priorTrendOf(w.Pretrend) {
				t.Fatalf("%s pattern %d: got %+v, reference kind %d formed %d breakout %d direction %d pretrend %d",
					c.Name, i, g, w.Kind, w.Formed, w.Breakout, w.Direction, w.Pretrend)
			}
			kindsSeen[g.Kind] = true
		}
	}
	for k := PatternHeadAndShouldersTop; k <= PatternDiamond; k++ {
		if !kindsSeen[k] {
			t.Errorf("the reference series never produce %v, so it is untested", k)
		}
	}
}

func TestFanBreaksMatchesTheReference(t *testing.T) {
	total := 0
	for _, c := range baseline(t) {
		got, err := FanBreaks(listOf(c.Close), baselinePivots(t, c), FanOptions{SegmentStarts: c.Breaks})
		if err != nil {
			t.Fatal(err)
		}
		// The reference lists the events of one row kind after another; order them the same way.
		want := make(map[[3]int]int)
		for _, f := range c.Fans {
			want[[3]int{f.Direction, f.Number, f.At}]++
		}
		if len(got) != len(c.Fans) {
			t.Fatalf("%s: %d fan breaks, reference has %d", c.Name, len(got), len(c.Fans))
		}
		for _, g := range got {
			d := -1
			if g.Direction == BreakUp {
				d = 1
			}
			key := [3]int{d, g.Line, g.Row}
			if want[key] == 0 {
				t.Fatalf("%s: fan break %+v is not in the reference", c.Name, g)
			}
			want[key]--
		}
		total += len(got)
	}
	if total == 0 {
		t.Fatal("no fan breaks to compare")
	}
}

func TestClassifyGapsMatchesTheReference(t *testing.T) {
	labels := map[string]struct {
		kind GapKind
		up   bool
	}{
		"CGU": {GapCommon, true}, "CGD": {GapCommon, false},
		"BGU_V": {GapBreakaway, true}, "BGU_N": {GapBreakaway, true}, "BGD_V": {GapBreakaway, false}, "BGD_N": {GapBreakaway, false},
		"RGU": {GapRunaway, true}, "RGD": {GapRunaway, false}, "EGU": {GapExhaustion, true}, "EGD": {GapExhaustion, false},
	}
	seen := map[[2]int]bool{}
	for _, c := range baseline(t) {
		got, err := ClassifyGaps(listOf(c.High), listOf(c.Low), listOf(c.Close), listOf(c.Volume), GapOptions{SegmentStarts: c.Breaks})
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != len(c.Gaps) {
			t.Fatalf("%s: %d gaps, reference has %d", c.Name, len(got), len(c.Gaps))
		}
		segEnd := func(row int) int {
			for _, b := range c.Breaks {
				if b > row {
					return b
				}
			}
			return len(c.Close)
		}
		fill := func(want *bool, row, window int) GapFill {
			if row+window >= segEnd(row) {
				return GapFillUnknown // the reference looks across a break; ClassifyGaps stops at it
			}
			if want == nil {
				t.Fatalf("%s row %d: the reference has no answer inside the segment", c.Name, row)
			}
			if *want {
				return GapFilled
			}
			return GapNotFilled
		}
		for i, w := range c.Gaps {
			g := got[i]
			wantKind, wantUp := GapUnclassified, w.Up
			if w.Label != "" {
				wantKind, wantUp = labels[w.Label].kind, labels[w.Label].up
			}
			if g.Row != w.Row || (g.Direction == BreakUp) != wantUp || g.Kind != wantKind {
				t.Fatalf("%s gap %d: got %+v, reference row %d up %v label %q", c.Name, i, g, w.Row, w.Up, w.Label)
			}
			if w.Label == "BGU_V" || w.Label == "BGD_V" || w.Label == "EGU" || w.Label == "EGD" {
				if !g.HighVolume {
					t.Fatalf("%s gap %d: %s is a heavy-volume class but HighVolume is false", c.Name, i, w.Label)
				}
			}
			if w.Label == "BGU_N" || w.Label == "BGD_N" {
				if g.HighVolume {
					t.Fatalf("%s gap %d: %s is a light-volume class but HighVolume is true", c.Name, i, w.Label)
				}
			}
			if d := g.TrendZ - w.TrendZ; d > 1e-8 || d < -1e-8 {
				t.Fatalf("%s gap %d: TrendZ %v, reference %v", c.Name, i, g.TrendZ, w.TrendZ)
			}
			if d := g.RelativeVolume - w.RelativeVolume; d > 1e-8 || d < -1e-8 {
				t.Fatalf("%s gap %d: RelativeVolume %v, reference %v", c.Name, i, g.RelativeVolume, w.RelativeVolume)
			}
			if f := fill(w.Fill5, w.Row, 5); g.Fills[0] != f {
				t.Fatalf("%s gap %d: Fills[0] %v, reference %v", c.Name, i, g.Fills[0], f)
			}
			if f := fill(w.Fill20, w.Row, 20); g.Fills[1] != f {
				t.Fatalf("%s gap %d: Fills[1] %v, reference %v", c.Name, i, g.Fills[1], f)
			}
			seen[[2]int{int(wantKind), b2i(wantUp)}] = true
		}
	}
	for _, l := range labels {
		if !seen[[2]int{int(l.kind), b2i(l.up)}] {
			t.Errorf("no %v (up=%v) in the reference series", l.kind, l.up)
		}
	}
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

func priorTrendOf(sign int) Trend {
	switch sign {
	case 1:
		return TrendUp
	case -1:
		return TrendDown
	}
	return TrendUnknown
}
