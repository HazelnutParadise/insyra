package ta

import (
	"math"
	"math/rand"
	"reflect"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

func candleLists(r *CandlePatternResult) []*insyra.DataList {
	return []*insyra.DataList{r.Doji, r.Hammer, r.HangingMan, r.InvertedHammer, r.ShootingStar, r.Engulfing, r.Harami,
		r.PiercingLine, r.DarkCloudCover, r.MorningStar, r.EveningStar, r.ThreeWhiteSoldiers, r.ThreeBlackCrows}
}

func candleTestSeries(n int, seed int64) (o, h, l, c []float64) {
	r := rand.New(rand.NewSource(seed))
	o, h, l, c = make([]float64, n), make([]float64, n), make([]float64, n), make([]float64, n)
	p := 100.0
	for i := 0; i < n; i++ {
		o[i] = p
		c[i] = p * (1 + 0.02*r.NormFloat64())
		hi, lo := math.Max(o[i], c[i]), math.Min(o[i], c[i])
		h[i] = hi * (1 + 0.03*r.Float64()*r.Float64())
		l[i] = lo * (1 - 0.03*r.Float64()*r.Float64())
		p = c[i]
	}
	return
}

func runCandleSlices(t *testing.T, o, h, l, c []float64, opts CandleOptions) *CandlePatternResult {
	t.Helper()
	res, err := CandlePatterns(prices(o...), prices(h...), prices(l...), prices(c...), opts)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

// Each segment is CandlePatterns of that segment alone: nothing looks back across a
// start, and a segment's first rows lack the history a pattern needs.
func TestCandleSegmentStartsEqualSeparateSegments(t *testing.T) {
	o, h, l, c := candleTestSeries(400, 3)
	for _, base := range []CandleOptions{{}, {TrendRows: 2, BodyRatio: 0.4}} {
		for _, starts := range [][]int{{100}, {50, 200}, {10, 11, 12, 300}, {399}, {200, 50, 200, 0}} {
			opts := base
			opts.SegmentStarts = starts
			got := candleLists(runCandleSlices(t, o, h, l, c, opts))
			clean, err := cleanSegmentStarts(starts, len(c))
			if err != nil {
				t.Fatal(err)
			}
			for k := range got {
				var want []any
				for _, b := range segmentBounds(len(c), clean) {
					seg := candleLists(runCandleSlices(t, o[b[0]:b[1]], h[b[0]:b[1]], l[b[0]:b[1]], c[b[0]:b[1]], base))
					want = append(want, seg[k].Data()...)
				}
				if !reflect.DeepEqual(got[k].Data(), want) {
					t.Fatalf("starts=%v base=%+v: list %s differs from separate segments", starts, base, got[k].GetName())
				}
			}
		}
	}
}

func TestCandleSegmentStartsDoNotReachBack(t *testing.T) {
	// A fall, then a hammer after the segment start: with the fall in the previous
	// segment the hammer has no trend of its own and must not be flagged.
	two := CandleOptions{TrendRows: 2}
	bars := append(falling(), bar{10, 10.2, 8, 10.1})
	whole := runCandles(t, two, bars...)
	if flagAt(t, whole.Hammer, 3) != 1 {
		t.Fatal("setup: the hammer must be flagged without a break")
	}
	two.SegmentStarts = []int{3}
	if got := runCandles(t, two, bars...); flagAt(t, got.Hammer, 3) != 0 {
		t.Error("a hammer must not read the fall from the previous segment")
	}
	// An engulfing pair split by a start is two segments of one bar each.
	pair := []bar{{12, 12.1, 9.9, 10}, {9.8, 12.5, 9.7, 12.4}}
	if flagAt(t, runCandles(t, CandleOptions{}, pair...).Engulfing, 1) != 1 {
		t.Fatal("setup: engulfing without a break")
	}
	if flagAt(t, runCandles(t, CandleOptions{SegmentStarts: []int{1}}, pair...).Engulfing, 1) != 0 {
		t.Error("engulfing must not use the bar before a segment start")
	}
}

func TestCandleSegmentStartsLeaveDefaultsAlone(t *testing.T) {
	o, h, l, c := candleTestSeries(300, 5)
	base := candleLists(runCandleSlices(t, o, h, l, c, CandleOptions{}))
	for _, starts := range [][]int{nil, {}, {0}} {
		got := candleLists(runCandleSlices(t, o, h, l, c, CandleOptions{SegmentStarts: starts}))
		for k := range got {
			if !reflect.DeepEqual(got[k].Data(), base[k].Data()) {
				t.Fatalf("starts %v changed %s", starts, got[k].GetName())
			}
		}
	}
	p := prices(10, 11, 12)
	for _, bad := range [][]int{{3}, {-1}} {
		if _, err := CandlePatterns(p, p, p, p, CandleOptions{SegmentStarts: bad}); err == nil {
			t.Errorf("start %v outside the rows must be an error", bad)
		}
	}
}

// LongShadowRatio and ShortShadowRatio are separate rules.
func TestCandleShadowRatiosAreSeparate(t *testing.T) {
	two := CandleOptions{TrendRows: 2}
	// body 0.1, lower shadow 2.0, upper shadow 0.4
	hammer := append(falling(), bar{10, 10.5, 8, 10.1})
	if flagAt(t, runCandles(t, two, hammer...).Hammer, 3) != 1 {
		t.Fatal("setup: a hammer with the defaults")
	}
	two.ShortShadowRatio = 10 // the upper shadow must be at most 2.0/10 = 0.2
	if flagAt(t, runCandles(t, two, hammer...).Hammer, 3) != 0 {
		t.Error("a stricter ShortShadowRatio must refuse the 0.4 upper shadow")
	}
	two = CandleOptions{TrendRows: 2, LongShadowRatio: 30} // the lower shadow must be at least 3.0
	if flagAt(t, runCandles(t, two, hammer...).Hammer, 3) != 0 {
		t.Error("a stricter LongShadowRatio must refuse the 2.0 lower shadow")
	}
	two = CandleOptions{TrendRows: 2, LongShadowRatio: 1.5, ShortShadowRatio: 2}
	if flagAt(t, runCandles(t, two, hammer...).Hammer, 3) != 1 {
		t.Error("a looser LongShadowRatio keeps the hammer")
	}
	// Soldiers: each upper shadow must be at most body / ShortShadowRatio.
	soldiers := []bar{{10, 11.2, 9.95, 11}, {10.8, 12.2, 10.7, 12}, {11.8, 13.2, 11.7, 13}} // upper 0.2, body 1
	if flagAt(t, runCandles(t, CandleOptions{}, soldiers...).ThreeWhiteSoldiers, 2) != 1 {
		t.Fatal("setup: soldiers with defaults")
	}
	if flagAt(t, runCandles(t, CandleOptions{ShortShadowRatio: 8}, soldiers...).ThreeWhiteSoldiers, 2) != 0 {
		t.Error("ShortShadowRatio 8 allows a soldier shadow of at most 1/8 of the body")
	}
	if flagAt(t, runCandles(t, CandleOptions{LongShadowRatio: 8}, soldiers...).ThreeWhiteSoldiers, 2) != 1 {
		t.Error("LongShadowRatio does not touch soldiers")
	}
	p := prices(10, 11, 12)
	for _, bad := range []CandleOptions{{LongShadowRatio: -1}, {ShortShadowRatio: -1}} {
		if _, err := CandlePatterns(p, p, p, p, bad); err == nil {
			t.Errorf("%+v must be an error", bad)
		}
	}
}
