package ta

import (
	"math"
	"math/rand"
	"reflect"
	"strings"
	"testing"
)

// volatility thresholds, checked against a two-pass standard deviation.
func TestVolatilityThresholdsByHand(t *testing.T) {
	p := []float64{100, 110, 99, 108.9}
	got, err := VolatilityThresholds(prices(p...), 2, VolatilityThresholdOptions{Window: 3, MinRows: 2, Floor: 0.01, WarmupVolatility: 0.05})
	if err != nil {
		t.Fatal(err)
	}
	sd := func(rows ...float64) float64 {
		var m float64
		for _, r := range rows {
			m += r
		}
		m /= float64(len(rows))
		var v float64
		for _, r := range rows {
			v += (r - m) * (r - m)
		}
		return math.Sqrt(v / float64(len(rows)))
	}
	r1, r2, r3 := math.Log(110.0/100), math.Log(99.0/110), math.Log(108.9/99)
	nearlyAll(t, "thresholds", got, []float64{0.1, 0.1, 2 * sd(r1, r2), 2 * sd(r1, r2, r3)})
}

func TestVolatilityThresholdsFloorWindowAndBreaks(t *testing.T) {
	flat := prices(100, 100, 100, 100, 100, 100)
	got, err := VolatilityThresholds(flat, 3, VolatilityThresholdOptions{MinRows: 2})
	if err != nil {
		t.Fatal(err)
	}
	nearlyAll(t, "floor", got, []float64{0.075, 0.075, 0.02, 0.02, 0.02, 0.02})

	// A break restarts the window: the first two rows after it are warm-up again.
	got, err = VolatilityThresholds(flat, 3, VolatilityThresholdOptions{MinRows: 2, SegmentStarts: []int{3}})
	if err != nil {
		t.Fatal(err)
	}
	nearlyAll(t, "break", got, []float64{0.075, 0.075, 0.02, 0.075, 0.075, 0.02})
}

func TestVolatilityThresholdsRefuseBadInput(t *testing.T) {
	good := prices(100, 101, 102)
	for name, c := range map[string]struct {
		multiple float64
		o        VolatilityThresholdOptions
	}{
		"zero multiple":     {0, VolatilityThresholdOptions{}},
		"negative multiple": {-1, VolatilityThresholdOptions{}},
		"NaN multiple":      {math.NaN(), VolatilityThresholdOptions{}},
		"negative window":   {1, VolatilityThresholdOptions{Window: -1}},
		"minrows>window":    {1, VolatilityThresholdOptions{Window: 5, MinRows: 6}},
		"break outside":     {1, VolatilityThresholdOptions{SegmentStarts: []int{3}}},
	} {
		if _, err := VolatilityThresholds(good, c.multiple, c.o); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
	if _, err := VolatilityThresholds(good, 1, VolatilityThresholdOptions{}, VolatilityThresholdOptions{}); err == nil {
		t.Error("two option sets should be an error")
	}
	if _, err := VolatilityThresholds(prices(100, 0, 101), 1); err == nil {
		t.Error("a zero price should be an error")
	}
}

func TestZigZagByWithAConstantThresholdIsZigZag(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	s := make([]float64, 300)
	s[0] = 100
	for i := 1; i < len(s); i++ {
		s[i] = s[i-1] * math.Exp(0.03*rng.NormFloat64())
	}
	th := make([]float64, len(s))
	for i := range th {
		th[i] = 0.08
	}
	want, _ := ZigZag(prices(s...), 0.08)
	got, err := ZigZagAdaptive(prices(s...), prices(th...))
	if err != nil {
		t.Fatal(err)
	}
	if len(want) == 0 || !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v\nwant %+v", got, want)
	}
}

// The threshold of a reversal is the one in force at the extreme, so a loose
// threshold at the peak lets a fall that a tight one would have confirmed pass.
func TestZigZagByUsesTheThresholdOfTheExtreme(t *testing.T) {
	p := prices(100, 120, 105, 130, 110)
	tight := prices(0.05, 0.05, 0.05, 0.05, 0.05)
	got, _ := ZigZagAdaptive(p, tight)
	if len(got) < 3 {
		t.Fatalf("with a tight threshold every swing counts: %+v", got)
	}
	loose := prices(0.05, 0.30, 0.05, 0.05, 0.05) // the peak at row 1 needs a 30% fall to confirm
	got, err := ZigZagAdaptive(p, loose)
	if err != nil {
		t.Fatal(err)
	}
	for _, pv := range got {
		if pv.Index == 1 {
			t.Fatalf("the peak at row 1 fell only 12.5%%, under its 30%% threshold: %+v", got)
		}
	}
}

func TestZigZagByIsCausalAndHonoursBreaks(t *testing.T) {
	rng := rand.New(rand.NewSource(5))
	s := make([]float64, 500)
	th := make([]float64, 500)
	s[0] = 100
	for i := 1; i < len(s); i++ {
		s[i] = s[i-1] * math.Exp(0.03*rng.NormFloat64())
	}
	for i := range th {
		th[i] = 0.05 + 0.1*rng.Float64()
	}
	breaks := []int{200, 350}
	full, err := ZigZagAdaptive(prices(s...), prices(th...), ZigZagOptions{SegmentStarts: breaks})
	if err != nil || len(full) < 5 {
		t.Fatalf("swings: %d, %v", len(full), err)
	}
	for _, pv := range full {
		seg := 0
		for _, b := range breaks {
			if pv.Index >= b {
				seg++
			}
		}
		if pv.Segment != seg {
			t.Fatalf("swing %+v is in segment %d", pv, seg)
		}
		if seg < len(breaks) && pv.ConfirmedAt >= breaks[seg] {
			t.Fatalf("swing %+v was confirmed after its segment ended", pv)
		}
	}
	for cut := 1; cut <= len(s); cut += 11 {
		var cutBreaks []int
		for _, b := range breaks {
			if b < cut {
				cutBreaks = append(cutBreaks, b)
			}
		}
		part, err := ZigZagAdaptive(prices(s[:cut]...), prices(th[:cut]...), ZigZagOptions{SegmentStarts: cutBreaks})
		if err != nil {
			t.Fatal(err)
		}
		var want []Swing
		for _, pv := range full {
			if pv.ConfirmedAt < cut {
				want = append(want, pv)
			}
		}
		if !reflect.DeepEqual(part, want) {
			t.Fatalf("cut at %d: got %+v, want %+v", cut, part, want)
		}
	}
}

func TestZigZagByRefusesBadInput(t *testing.T) {
	p := prices(100, 110, 100)
	for name, th := range map[string]*struct {
		l    []float64
		opts []ZigZagOptions
	}{
		"short":        {[]float64{0.1, 0.1}, nil},
		"zero":         {[]float64{0.1, 0, 0.1}, nil},
		"one":          {[]float64{0.1, 1, 0.1}, nil},
		"break out":    {[]float64{0.1, 0.1, 0.1}, []ZigZagOptions{{SegmentStarts: []int{3}}}},
		"negative brk": {[]float64{0.1, 0.1, 0.1}, []ZigZagOptions{{SegmentStarts: []int{-1}}}},
		"two options":  {[]float64{0.1, 0.1, 0.1}, []ZigZagOptions{{}, {}}},
	} {
		if _, err := ZigZagAdaptive(p, prices(th.l...), th.opts...); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
	if _, err := ZigZagAdaptive(p, nil); err == nil {
		t.Error("nil thresholds should be an error")
	}
}

// series of n rows at 100 with the given rows overridden.
func flatWith(n int, at map[int]float64) []float64 {
	c := make([]float64, n)
	for i := range c {
		c[i] = 100
	}
	for i, v := range at {
		c[i] = v
	}
	return c
}

func pv(kind SwingKind, index int, price float64, confirmed int) Swing {
	return Swing{kind, index, price, confirmed, 0}
}

func findPattern(ps []Pattern, kind PatternKind) []Pattern {
	var out []Pattern
	for _, p := range ps {
		if p.Kind == kind {
			out = append(out, p)
		}
	}
	return out
}

var doubleTop = []Swing{pv(SwingLow, 0, 80, 1), pv(SwingHigh, 10, 100, 12), pv(SwingLow, 20, 85, 22), pv(SwingHigh, 30, 99, 32)}

func TestDoubleTopBreaksDownThroughTheTrough(t *testing.T) {
	c := flatWith(60, nil)
	c[36] = 84 // under the 85 trough
	got, err := DetectPatterns(prices(c...), doubleTop)
	if err != nil {
		t.Fatal(err)
	}
	dt := findPattern(got, PatternDoubleTop)
	want := []Pattern{{Kind: PatternDoubleTop, Start: 10, ConfirmedAt: 32, ExpiresAt: 50, BreakoutAt: 36, Direction: BreakDown}}
	if !reflect.DeepEqual(dt, want) {
		t.Fatalf("got %+v\nwant %+v", dt, want)
	}
}

func TestDoubleTopIsInvalidatedByANewHigh(t *testing.T) {
	c := flatWith(60, map[int]float64{35: 101, 36: 84})
	got, _ := DetectPatterns(prices(c...), doubleTop)
	dt := findPattern(got, PatternDoubleTop)
	if len(dt) != 1 || dt[0].BreakoutAt != -1 || dt[0].Direction != BreakNone {
		t.Fatalf("a close above the peaks before the break cancels the pattern: %+v", dt)
	}
}

func TestDoubleTopExpires(t *testing.T) {
	c := flatWith(80, map[int]float64{55: 84}) // expires at 50
	got, _ := DetectPatterns(prices(c...), doubleTop)
	dt := findPattern(got, PatternDoubleTop)
	if len(dt) != 1 || dt[0].BreakoutAt != -1 {
		t.Fatalf("a break after the expiry does not count: %+v", dt)
	}
}

func TestPatternToleranceOptions(t *testing.T) {
	c := flatWith(60, nil)
	def, _ := DetectPatterns(prices(c...), doubleTop)
	if len(findPattern(def, PatternAscendingTriangle)) != 1 || len(findPattern(def, PatternRectangle)) != 0 {
		t.Fatalf("the lows rise 25%% of the height, which is not flat by default: %+v", def)
	}
	wide, err := DetectPatterns(prices(c...), doubleTop, PatternOptions{FlatTolerance: 0.3})
	if err != nil {
		t.Fatal(err)
	}
	if len(findPattern(wide, PatternRectangle)) != 1 {
		t.Fatalf("with FlatTolerance 0.3 it is a box: %+v", wide)
	}
	tight, _ := DetectPatterns(prices(c...), doubleTop, PatternOptions{SymmetryTolerance: 0.01})
	if len(findPattern(tight, PatternDoubleTop)) != 0 {
		t.Fatalf("peaks 1 apart are not equal within 1%% of the height: %+v", tight)
	}
	near, _ := DetectPatterns(prices(c...), doubleTop, PatternOptions{MinDoubleSeparation: 21})
	if len(findPattern(near, PatternDoubleTop)) != 0 {
		t.Fatalf("peaks 20 rows apart are too close for a minimum of 21: %+v", near)
	}
	short, _ := DetectPatterns(prices(c...), doubleTop, PatternOptions{MaxRows: 5})
	dt := findPattern(short, PatternDoubleTop)
	if len(dt) != 1 || dt[0].ExpiresAt != 35 {
		t.Fatalf("MaxRows 5 should expire at row 35: %+v", dt)
	}
}

var headAndShoulders = []Swing{
	pv(SwingLow, 0, 80, 2), pv(SwingHigh, 10, 100, 12), pv(SwingLow, 20, 90, 22),
	pv(SwingHigh, 30, 110, 32), pv(SwingLow, 40, 91, 42), pv(SwingHigh, 50, 99, 52),
}

func TestHeadAndShouldersBreaksThroughTheNeckline(t *testing.T) {
	c := flatWith(80, map[int]float64{54: 91.5, 56: 88}) // neckline is 90 + (t-20)/20 = 91.75 at row 55, 91.8 at 56
	got, err := DetectPatterns(prices(c...), headAndShoulders)
	if err != nil {
		t.Fatal(err)
	}
	hs := findPattern(got, PatternHeadAndShouldersTop)
	// Row 52 closes at 100, above the neckline, so the first close under it is row 54.
	want := []Pattern{{Kind: PatternHeadAndShouldersTop, Start: 10, ConfirmedAt: 52, ExpiresAt: 90, BreakoutAt: 54, Direction: BreakDown}}
	if !reflect.DeepEqual(hs, want) {
		t.Fatalf("got %+v\nwant %+v", hs, want)
	}
}

func TestHeadAndShouldersBottomIsTheMirror(t *testing.T) {
	var mirrored []Swing
	for _, p := range headAndShoulders {
		k := SwingLow
		if p.Kind == SwingLow {
			k = SwingHigh
		}
		mirrored = append(mirrored, pv(k, p.Index, 200-p.Price, p.ConfirmedAt))
	}
	c := flatWith(80, map[int]float64{54: 200 - 91.5})
	got, err := DetectPatterns(prices(c...), mirrored)
	if err != nil {
		t.Fatal(err)
	}
	hs := findPattern(got, PatternHeadAndShouldersBottom)
	if len(hs) != 1 || hs[0].Direction != BreakUp || hs[0].BreakoutAt != 54 {
		t.Fatalf("got %+v", hs)
	}
	if len(findPattern(got, PatternHeadAndShouldersTop)) != 0 {
		t.Error("the mirror must not also be a top")
	}
}

func TestHeadAndShouldersNeedsAHigherHead(t *testing.T) {
	flat := append([]Swing(nil), headAndShoulders...)
	flat[3] = pv(SwingHigh, 30, 99.5, 32) // the middle peak is no longer the highest
	got, _ := DetectPatterns(prices(flatWith(80, nil)...), flat)
	if len(findPattern(got, PatternHeadAndShouldersTop)) != 0 {
		t.Fatalf("no head, no pattern: %+v", got)
	}
}

func TestPatternsDoNotCrossASegment(t *testing.T) {
	split := append([]Swing(nil), doubleTop...)
	for i := 2; i < 4; i++ {
		split[i].Segment = 1
	}
	got, err := DetectPatterns(prices(flatWith(60, nil)...), split, PatternOptions{SegmentStarts: []int{15}})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("swings from two segments make no pattern: %+v", got)
	}
}

func TestDiamondBreaksDown(t *testing.T) {
	// widening: peak 100 -> 110, trough 90 -> 85; narrowing: peak 105, trough 92
	d := []Swing{
		pv(SwingLow, 0, 80, 1), pv(SwingHigh, 10, 100, 12), pv(SwingLow, 20, 90, 22), pv(SwingHigh, 30, 110, 32),
		pv(SwingLow, 40, 85, 42), pv(SwingHigh, 50, 105, 52), pv(SwingLow, 60, 92, 62),
	}
	c := flatWith(100, map[int]float64{66: 80})
	got, err := DetectPatterns(prices(c...), d)
	if err != nil {
		t.Fatal(err)
	}
	di := findPattern(got, PatternDiamond)
	if len(di) != 1 || di[0].Direction != BreakDown || di[0].BreakoutAt != 66 || di[0].Start != 10 {
		t.Fatalf("got %+v", di)
	}
}

// A pattern found on a series cut short is the pattern of the full series
// with whatever happened after the cut forgotten.
func TestDetectPatternsIsCausal(t *testing.T) {
	rng := rand.New(rand.NewSource(17))
	n := 700
	s := make([]float64, n)
	s[0] = 100
	for i := 1; i < n; i++ {
		s[i] = s[i-1] * math.Exp(0.015*rng.NormFloat64()+0.01*math.Sin(float64(i)/9))
	}
	th, _ := VolatilityThresholds(prices(s...), 1.5)
	swings, _ := ZigZagAdaptive(prices(s...), th)
	full, err := DetectPatterns(prices(s...), swings)
	if err != nil || len(full) < 20 {
		t.Fatalf("patterns: %d, %v", len(full), err)
	}
	for cut := 30; cut <= n; cut += 13 {
		sub := th.Data()[:cut]
		cp, _ := ZigZagAdaptive(prices(s[:cut]...), prices(toFloats(sub)...))
		part, err := DetectPatterns(prices(s[:cut]...), cp)
		if err != nil {
			t.Fatal(err)
		}
		var want []Pattern
		for _, p := range full {
			if p.ConfirmedAt < cut {
				if p.BreakoutAt >= cut {
					p.BreakoutAt, p.Direction = -1, BreakNone
				}
				want = append(want, p)
			}
		}
		if len(part) != len(want) || (len(want) > 0 && !reflect.DeepEqual(part, want)) {
			t.Fatalf("cut at %d: %d patterns, want %d", cut, len(part), len(want))
		}
	}
}

func toFloats(v []any) []float64 {
	out := make([]float64, len(v))
	for i, x := range v {
		out[i] = x.(float64)
	}
	return out
}

func TestDetectPatternsRefusesBadInput(t *testing.T) {
	p := prices(flatWith(60, nil)...)
	for name, swings := range map[string][]Swing{
		"confirmed before it happened": {pv(SwingLow, 10, 90, 5)},
		"outside the series":           {pv(SwingLow, 10, 90, 60)},
		"out of order":                 {pv(SwingLow, 10, 90, 12), pv(SwingHigh, 5, 100, 14)},
	} {
		if _, err := DetectPatterns(p, swings); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
	if _, err := DetectPatterns(p, nil, PatternOptions{MaxRows: -1}); err == nil {
		t.Error("a negative option should be an error")
	}
	if _, err := DetectPatterns(p, nil, PatternOptions{}, PatternOptions{}); err == nil {
		t.Error("two option sets should be an error")
	}
	if _, err := DetectPatterns(nil, nil); err == nil {
		t.Error("nil prices should be an error")
	}
	if got, err := DetectPatterns(p, nil); err != nil || len(got) != 0 {
		t.Errorf("no swings, no patterns: %v, %v", got, err)
	}
}

// A fan from a rising trough: each close under the line through the next
// higher trough is a break, and the third ends it.
func TestFanBreaksCountsThreeBreaks(t *testing.T) {
	lows := []Swing{pv(SwingLow, 0, 100, 2), pv(SwingLow, 20, 110, 22), pv(SwingLow, 60, 120, 62), pv(SwingLow, 100, 128, 102)}
	c := flatWith(130, nil)
	for i := range c {
		c[i] = 200
	}
	c[30], c[70], c[110] = 114, 120, 125 // under 115, 123.3 and 130.8
	got, err := FanBreaks(prices(c...), lows)
	if err != nil {
		t.Fatal(err)
	}
	want := []FanBreak{
		{BreakDown, 1, 30, 0, 20}, {BreakDown, 2, 70, 0, 60}, {BreakDown, 3, 110, 0, 100},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v\nwant %+v", got, want)
	}
}

func TestFanIsDiscardedWhenThePricePassesTheAnchor(t *testing.T) {
	lows := []Swing{pv(SwingLow, 0, 100, 2), pv(SwingLow, 20, 110, 22)}
	c := make([]float64, 60)
	for i := range c {
		c[i] = 200
	}
	c[25] = 99  // under the anchor: the fan is dropped, not broken
	c[30] = 114 // would have broken the line
	got, _ := FanBreaks(prices(c...), lows)
	if len(got) != 0 {
		t.Fatalf("got %+v", got)
	}
}

// Regimes never join swings from different data segments: a high ending one
// segment and a high starting the next is not an alternation error, and a high
// followed by a low across a break is not a bear stretch.
func TestRegimesStayInsideTheirSegment(t *testing.T) {
	swings := []Swing{
		{SwingLow, 0, 100, 2, 0}, {SwingHigh, 10, 120, 12, 0},
		{SwingHigh, 30, 90, 32, 1}, {SwingLow, 40, 80, 42, 1}, {SwingHigh, 50, 95, 52, 1},
	}
	got, err := Regimes(swings)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0].Kind != RegimeBull || got[0].End != 10 || got[1].Kind != RegimeBear || got[1].Start != 30 || got[2].Kind != RegimeBull {
		t.Fatalf("got %+v", got)
	}
	for _, r := range got {
		if r.Start < 30 && r.End >= 30 {
			t.Fatalf("regime %+v crosses the break", r)
		}
	}
	// Inside one segment the alternation check still applies.
	if _, err := Regimes([]Swing{{SwingLow, 0, 1, 1, 0}, {SwingLow, 2, 1, 3, 0}}); err == nil {
		t.Error("two lows in one segment should be an error")
	}
	if _, err := Regimes([]Swing{{SwingLow, 0, 1, 1, 1}, {SwingHigh, 2, 1, 3, 0}}); err == nil {
		t.Error("segments out of order should be an error")
	}
	// MinPhase applies inside a segment only.
	short := []Swing{{SwingLow, 0, 100, 1, 0}, {SwingHigh, 1, 101, 2, 0}, {SwingHigh, 30, 90, 31, 1}, {SwingLow, 40, 80, 41, 1}}
	got, err = Regimes(short, RegimeOptions{MinPhase: 3})
	if err != nil || len(got) != 1 || got[0].Kind != RegimeBear {
		t.Fatalf("the one-row bull stretch in segment 0 is removed, the bear stretch in segment 1 stays: %+v, %v", got, err)
	}
}

func TestPatternsRefuseBreaksThatDifferFromTheSwings(t *testing.T) {
	c := prices(flatWith(60, nil)...)
	// doubleTop was made without breaks, so every swing is in segment 0.
	if _, err := DetectPatterns(c, doubleTop, PatternOptions{SegmentStarts: []int{15}}); err == nil || !strings.Contains(err.Error(), "Segment") {
		t.Errorf("DetectPatterns: %v", err)
	}
	if _, err := FanBreaks(c, doubleTop, FanOptions{SegmentStarts: []int{15}}); err == nil {
		t.Error("FanBreaks should refuse it too")
	}
	if _, err := FanBreaks(c, doubleTop, FanOptions{}, FanOptions{}); err == nil {
		t.Error("two option sets should be an error")
	}
}

func TestPriorTrendIsSetForEveryPatternKind(t *testing.T) {
	d := []Swing{
		pv(SwingLow, 0, 80, 1), pv(SwingHigh, 10, 100, 12), pv(SwingLow, 20, 90, 22), pv(SwingHigh, 30, 110, 32),
		pv(SwingLow, 40, 85, 42), pv(SwingHigh, 50, 105, 52), pv(SwingLow, 60, 92, 62),
	}
	c := flatWith(100, map[int]float64{5: 90}) // the close on the pattern's first row, 100, is above the 90 five rows before
	opts := PatternOptions{TrendRows: 5}
	for name, tc := range map[string]struct {
		swings []Swing
		kind   PatternKind
	}{
		"double top":         {doubleTop, PatternDoubleTop},
		"head and shoulders": {headAndShoulders, PatternHeadAndShouldersTop},
		"diamond":            {d, PatternDiamond},
	} {
		got, err := DetectPatterns(prices(c...), tc.swings, opts)
		if err != nil {
			t.Fatal(err)
		}
		ps := findPattern(got, tc.kind)
		if len(ps) == 0 {
			t.Fatalf("%s: not found", name)
		}
		for _, p := range ps {
			if p.PriorTrend != TrendUp {
				t.Errorf("%s: PriorTrend %v, want TrendUp", name, p.PriorTrend)
			}
		}
	}
}

func TestBreakoutCanBeOnTheConfirmingRow(t *testing.T) {
	// Row 32 confirms the second peak and closes under the 85 trough: that close is known on row 32,
	// so the break is dated there.
	c := flatWith(60, map[int]float64{32: 84})
	got, err := DetectPatterns(prices(c...), doubleTop)
	if err != nil {
		t.Fatal(err)
	}
	dt := findPattern(got, PatternDoubleTop)
	if len(dt) != 1 || dt[0].BreakoutAt != 32 || dt[0].Direction != BreakDown {
		t.Fatalf("got %+v", dt)
	}
}
