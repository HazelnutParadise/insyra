package ta

import (
	"math"
	"testing"
)

// bars returns 100 calm bars: close 100 or 100.5, range 99 to 101, volume 1000.
func calmBars() (h, l, c, v []float64) {
	n := 100
	h, l, c, v = make([]float64, n), make([]float64, n), make([]float64, n), make([]float64, n)
	for i := range c {
		h[i], l[i], v[i] = 101, 99, 1000
		c[i] = 100 + 0.5*float64(i%2)
	}
	return
}

func TestClassifyGapsBreakawayOnHeavyVolumeAndItsFill(t *testing.T) {
	h, l, c, v := calmBars()
	// Row 65 gaps up out of the range on three times the volume; the price
	// comes back to the old high (101) on row 68.
	h[65], l[65], c[65], v[65] = 108, 105, 106, 3000
	for i := 66; i < 100; i++ {
		h[i], l[i], c[i] = 107, 105, 106
	}
	l[68] = 100.5
	got, err := ClassifyGaps(prices(h...), prices(l...), prices(c...), prices(v...))
	if err != nil {
		t.Fatal(err)
	}
	// The drop back to the range on row 66 is a gap down only if it leaves a gap: it does not.
	if len(got) != 1 {
		t.Fatalf("got %+v", got)
	}
	g := got[0]
	if g.Row != 65 || g.Direction != BreakUp || g.Kind != GapBreakaway || !g.HighVolume {
		t.Fatalf("got %+v", g)
	}
	if math.Abs(g.RelativeVolume-3) > 1e-12 || g.TrendZ != 0 {
		t.Errorf("relative volume %v, trend %v", g.RelativeVolume, g.TrendZ)
	}
	if g.Fills[0] != GapFilled || g.Fills[1] != GapFilled {
		t.Errorf("fills %v %v", g.Fills[0], g.Fills[1])
	}
}

func TestClassifyGapsCommonGapDownAndFillsNotYetKnown(t *testing.T) {
	h, l, c, v := calmBars()
	l[70] = 95 // a dip inside the lookback, so the gap close stays inside the range
	// Row 80 gaps down from a previous low of 99 but closes at 97, inside 95..101.
	h[80], l[80], c[80] = 98.5, 96, 97
	for i := 81; i < 100; i++ {
		h[i], l[i], c[i] = 98, 96, 97
	}
	got, err := ClassifyGaps(prices(h...), prices(l...), prices(c...), prices(v...))
	if err != nil {
		t.Fatal(err)
	}
	// Row 81 opens inside row 80's range, so only row 80 is a gap.
	if len(got) != 1 || got[0].Row != 80 || got[0].Direction != BreakDown || got[0].Kind != GapCommon {
		t.Fatalf("got %+v", got)
	}
	if got[0].Fills[0] != GapNotFilled || got[0].Fills[1] != GapFillUnknown {
		t.Errorf("fills %v %v: the 20-row window runs past the data", got[0].Fills[0], got[0].Fills[1])
	}
}

// trendBars returns 140 bars that drift by d a day with a 0.4% alternating
// wobble and a wide range, so nothing gaps, then a gap up at row 80 to a new
// high on the given volume (the others trade 1000).
func trendBars(d, gapVolume float64) (h, l, c, v []float64) {
	n := 140
	h, l, c, v = make([]float64, n), make([]float64, n), make([]float64, n), make([]float64, n)
	c[0] = 100
	for i := range c {
		if i > 0 {
			wobble := 0.004
			if i%2 == 1 {
				wobble = -wobble
			}
			c[i] = c[i-1] * (1 + d + wobble)
		}
		h[i], l[i], v[i] = c[i]*1.02, c[i]*0.98, 1000
		if i == 80 {
			l[i] = h[i-1] * 1.01
			c[i] = l[i] * 1.01
			h[i] = c[i] * 1.01
			v[i] = gapVolume
		}
	}
	return
}

func TestClassifyGapsRunawayExhaustionAndUnclassified(t *testing.T) {
	cases := []struct {
		name           string
		drift, volume  float64
		zLow, zHigh    float64
		wantKind       GapKind
		wantHighVolume bool
	}{
		{"no trend", 0, 1000, -0.5, 0.5, GapBreakaway, false},
		{"mid trend", 0.002, 1000, 1, 2, GapRunaway, false},
		{"long trend on heavy volume", 0.0035, 4000, 2, 100, GapExhaustion, true},
		{"long trend on light volume", 0.0035, 1000, 2, 100, GapUnclassified, false},
		{"fall then a gap up stays in the range", -0.0035, 1000, -100, -1, GapCommon, false},
	}
	for _, tc := range cases {
		h, l, c, v := trendBars(tc.drift, tc.volume)
		got, err := ClassifyGaps(prices(h...), prices(l...), prices(c...), prices(v...))
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0].Row != 80 {
			t.Fatalf("%s: the only gap should be on row 80, got %d gaps", tc.name, len(got))
		}
		g := got[0]
		if g.TrendZ < tc.zLow || g.TrendZ >= tc.zHigh {
			t.Fatalf("%s: TrendZ %v is outside [%v, %v), so the fixture no longer tests this class", tc.name, g.TrendZ, tc.zLow, tc.zHigh)
		}
		if g.Kind != tc.wantKind || g.HighVolume != tc.wantHighVolume {
			t.Errorf("%s: kind %v and heavy %v, want %v and %v (z=%v)", tc.name, g.Kind, g.HighVolume, tc.wantKind, tc.wantHighVolume, g.TrendZ)
		}
	}

	// Options move the boundaries: a runaway needs a bigger move, or a lighter definition of heavy volume.
	h, l, c, v := trendBars(0.002, 1000)
	strict, err := ClassifyGaps(prices(h...), prices(l...), prices(c...), prices(v...), GapOptions{RunawayZ: 3, ExhaustionZ: 4})
	if err != nil || strict[0].Kind != GapBreakaway {
		t.Errorf("with TrendZ 3 a move of 1.5 is small enough to be a breakaway: %+v, %v", strict, err)
	}
	h, l, c, v = trendBars(0.0035, 1600)
	loose, err := ClassifyGaps(prices(h...), prices(l...), prices(c...), prices(v...), GapOptions{HighVolumeRatio: 1.2})
	if err != nil || loose[0].Kind != GapExhaustion {
		t.Errorf("1.6 times the volume is heavy when the bar is 1.2: %+v, %v", loose, err)
	}
}

func TestClassifyGapsNeedsAFullSegmentBehindIt(t *testing.T) {
	h, l, c, v := calmBars()
	h[65], l[65], c[65], v[65] = 108, 105, 106, 3000
	for i := 66; i < 100; i++ {
		h[i], l[i], c[i] = 107, 105, 106
	}
	got, err := ClassifyGaps(prices(h...), prices(l...), prices(c...), prices(v...), GapOptions{SegmentStarts: []int{30}})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("row 65 is only 35 rows into its segment: %+v", got)
	}
	// A fill window that crosses a break is unknown.
	got, _ = ClassifyGaps(prices(h...), prices(l...), prices(c...), prices(v...), GapOptions{SegmentStarts: []int{68}})
	if len(got) != 1 || got[0].Fills[0] != GapFillUnknown {
		t.Fatalf("got %+v", got)
	}
}

func TestClassifyGapsRefusesBadInput(t *testing.T) {
	h, l, c, v := calmBars()
	if _, err := ClassifyGaps(prices(h...), prices(l[:50]...), prices(c...), prices(v...)); err == nil {
		t.Error("different lengths should be an error")
	}
	h2 := append([]float64(nil), h...)
	h2[5] = 98 // below the low
	if _, err := ClassifyGaps(prices(h2...), prices(l...), prices(c...), prices(v...)); err == nil {
		t.Error("a high below the low should be an error")
	}
	l3 := append([]float64(nil), l...)
	l3[5] = 0 // a zero low would invent a gap
	if _, err := ClassifyGaps(prices(h...), prices(l3...), prices(c...), prices(v...)); err == nil {
		t.Error("a low that is not positive should be an error")
	}
	h3 := append([]float64(nil), h...)
	h3[5] = -1
	if _, err := ClassifyGaps(prices(h3...), prices(l...), prices(c...), prices(v...)); err == nil {
		t.Error("a high that is not positive should be an error")
	}
	v2 := append([]float64(nil), v...)
	v2[5] = -1
	if _, err := ClassifyGaps(prices(h...), prices(l...), prices(c...), prices(v2...)); err == nil {
		t.Error("negative volume should be an error")
	}
	if _, err := ClassifyGaps(prices(h...), prices(l...), prices(c...), prices(v...), GapOptions{RunawayZ: 3, ExhaustionZ: 2}); err == nil {
		t.Error("ExhaustionZ below TrendZ should be an error")
	}
	if _, err := ClassifyGaps(prices(h...), prices(l...), prices(c...), prices(v...), GapOptions{LookbackRows: -1}); err == nil {
		t.Error("a negative option should be an error")
	}
	if _, err := ClassifyGaps(nil, prices(l...), prices(c...), prices(v...)); err == nil {
		t.Error("nil high should be an error")
	}
	got, err := ClassifyGaps(prices(), prices(), prices(), prices())
	if err != nil || len(got) != 0 {
		t.Errorf("empty input: %v, %v", got, err)
	}
}

func TestClassifyGapsFillWindowsAreOptions(t *testing.T) {
	h, l, c, v := calmBars()
	h[65], l[65], c[65], v[65] = 108, 105, 106, 3000
	for i := 66; i < 100; i++ {
		h[i], l[i], c[i] = 107, 105, 106
	}
	l[70] = 100.5 // back to the old high 5 rows after the gap
	got, err := ClassifyGaps(prices(h...), prices(l...), prices(c...), prices(v...), GapOptions{FillWindows: []int{1, 4, 10}})
	if err != nil || len(got) != 1 {
		t.Fatalf("%v, %v", got, err)
	}
	if want := []GapFill{GapNotFilled, GapNotFilled, GapFilled}; len(got[0].Fills) != 3 || got[0].Fills[0] != want[0] || got[0].Fills[1] != want[1] || got[0].Fills[2] != want[2] {
		t.Errorf("fills %v, want %v", got[0].Fills, want)
	}
	if _, err := ClassifyGaps(prices(h...), prices(l...), prices(c...), prices(v...), GapOptions{FillWindows: []int{0}}); err == nil {
		t.Error("a window of 0 should be an error")
	}
	none, err := ClassifyGaps(prices(h...), prices(l...), prices(c...), prices(v...), GapOptions{FillWindows: []int{}})
	if err != nil || len(none) != 1 || len(none[0].Fills) != 0 {
		t.Errorf("an empty list asks for no fills: %v, %v", none, err)
	}
}
