package ta

import (
	"testing"

	"github.com/HazelnutParadise/insyra"
)

type bar [4]float64 // open, high, low, close

func runCandles(t *testing.T, opts CandleOptions, bars ...bar) *CandlePatternResult {
	t.Helper()
	o, h, l, c := make([]float64, len(bars)), make([]float64, len(bars)), make([]float64, len(bars)), make([]float64, len(bars))
	for i, b := range bars {
		o[i], h[i], l[i], c[i] = b[0], b[1], b[2], b[3]
	}
	res, err := CandlePatterns(prices(o...), prices(h...), prices(l...), prices(c...), opts)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func flagAt(t *testing.T, l *insyra.DataList, row int) int {
	t.Helper()
	v := l.Data()[row]
	if f, ok := v.(int); ok {
		return f
	}
	t.Fatalf("flag is %T, want int", v)
	return 0
}

// the lead-in that makes a trend: three flat-bodied bars at falling or rising closes
func falling() []bar { return []bar{{12, 12.1, 11.9, 12}, {11, 11.1, 10.9, 11}, {10, 10.1, 9.9, 10}} }
func rising() []bar  { return []bar{{10, 10.1, 9.9, 10}, {11, 11.1, 10.9, 11}, {12, 12.1, 11.9, 12}} }

func TestCandleSingleBarPatterns(t *testing.T) {
	two := CandleOptions{TrendRows: 2}
	doji := runCandles(t, two, bar{10, 11, 9, 10.1})
	if flagAt(t, doji.Doji, 0) != 1 {
		t.Error("a body of 5% of the range is a doji")
	}
	hammerBar := bar{10, 10.2, 8, 10.1}
	res := runCandles(t, two, append(falling(), hammerBar)...)
	if flagAt(t, res.Hammer, 3) != 1 || flagAt(t, res.HangingMan, 3) != 0 {
		t.Error("a small body with a long lower shadow after a fall is a hammer")
	}
	res = runCandles(t, two, append(rising(), hammerBar)...)
	if flagAt(t, res.HangingMan, 3) != -1 || flagAt(t, res.Hammer, 3) != 0 {
		t.Error("the same bar after a rise is a hanging man")
	}
	starBar := bar{10, 12, 9.9, 10.1}
	res = runCandles(t, two, append(falling(), starBar)...)
	if flagAt(t, res.InvertedHammer, 3) != 1 || flagAt(t, res.ShootingStar, 3) != 0 {
		t.Error("a long upper shadow after a fall is an inverted hammer")
	}
	res = runCandles(t, two, append(rising(), starBar)...)
	if flagAt(t, res.ShootingStar, 3) != -1 {
		t.Error("a long upper shadow after a rise is a shooting star")
	}
	// Without enough rows to read the trend there is no hammer.
	if res := runCandles(t, CandleOptions{}, hammerBar); flagAt(t, res.Hammer, 0) != 0 {
		t.Error("no trend can be read from one row")
	}
	// A plain bar is none of these.
	plain := runCandles(t, two, append(falling(), bar{10, 11, 9.5, 10.8})...)
	for _, l := range []*insyra.DataList{plain.Doji, plain.Hammer, plain.InvertedHammer} {
		if flagAt(t, l, 3) != 0 {
			t.Errorf("a plain bar was flagged in %s", l.GetName())
		}
	}
}

func TestCandleTwoBarPatterns(t *testing.T) {
	o := CandleOptions{}
	bull := runCandles(t, o, bar{10.5, 10.6, 9.4, 9.5}, bar{9.4, 11.1, 9.3, 11})
	if flagAt(t, bull.Engulfing, 1) != 1 {
		t.Error("a rise that engulfs the previous fall is a bullish engulfing")
	}
	bear := runCandles(t, o, bar{9.5, 10.6, 9.4, 10.5}, bar{10.6, 10.7, 8.9, 9})
	if flagAt(t, bear.Engulfing, 1) != -1 {
		t.Error("a fall that engulfs the previous rise is a bearish engulfing")
	}
	har := runCandles(t, o, bar{12, 12.1, 9.9, 10}, bar{10.6, 10.9, 10.4, 10.8})
	if flagAt(t, har.Harami, 1) != 1 {
		t.Error("a small body inside a long fall is a bullish harami")
	}
	har = runCandles(t, o, bar{10, 12.1, 9.9, 12}, bar{11.4, 11.6, 11.1, 11.2})
	if flagAt(t, har.Harami, 1) != -1 {
		t.Error("a small body inside a long rise is a bearish harami")
	}
	pier := runCandles(t, o, bar{12, 12.1, 9.9, 10}, bar{9.5, 11.6, 9.4, 11.5})
	if flagAt(t, pier.PiercingLine, 1) != 1 {
		t.Error("opening below the low and closing past the midpoint is a piercing line")
	}
	dark := runCandles(t, o, bar{10, 12.1, 9.9, 12}, bar{12.5, 12.6, 10.4, 10.5})
	if flagAt(t, dark.DarkCloudCover, 1) != -1 {
		t.Error("opening above the high and closing under the midpoint is a dark cloud cover")
	}
	// An engulfing needs a bigger body: the same colours with a smaller one are not.
	no := runCandles(t, o, bar{10.5, 10.6, 9.4, 9.5}, bar{9.6, 10.1, 9.5, 10})
	if flagAt(t, no.Engulfing, 1) != 0 {
		t.Error("a smaller body does not engulf")
	}
}

func TestCandleThreeBarPatterns(t *testing.T) {
	o := CandleOptions{}
	star := runCandles(t, o, bar{12, 12.1, 9.9, 10}, bar{9.4, 9.7, 9.1, 9.5}, bar{9.6, 11.6, 9.5, 11.5})
	if flagAt(t, star.MorningStar, 2) != 1 {
		t.Error("morning star")
	}
	eve := runCandles(t, o, bar{10, 12.1, 9.9, 12}, bar{12.5, 12.9, 12.3, 12.6}, bar{12.4, 12.5, 10.4, 10.5})
	if flagAt(t, eve.EveningStar, 2) != -1 {
		t.Error("evening star")
	}
	soldiers := runCandles(t, o, bar{10, 11.1, 9.95, 11}, bar{10.8, 12.1, 10.7, 12}, bar{11.8, 13.1, 11.7, 13})
	if flagAt(t, soldiers.ThreeWhiteSoldiers, 2) != 1 {
		t.Error("three white soldiers")
	}
	crows := runCandles(t, o, bar{13, 13.05, 11.9, 12}, bar{12.2, 12.25, 10.9, 11}, bar{11.2, 11.25, 9.9, 10})
	if flagAt(t, crows.ThreeBlackCrows, 2) != -1 {
		t.Error("three black crows")
	}
	// Soldiers with a long upper shadow are not soldiers.
	wick := runCandles(t, o, bar{10, 12, 9.95, 11}, bar{10.8, 12.1, 10.7, 12}, bar{11.8, 13.1, 11.7, 13})
	if flagAt(t, wick.ThreeWhiteSoldiers, 2) != 0 {
		t.Error("a long upper shadow disqualifies a soldier")
	}
}

func TestCandleOptionsChangeTheCutoffs(t *testing.T) {
	b := bar{10, 11, 9, 10.5} // body 0.5 of range 2: not a doji at 0.1
	if flagAt(t, runCandles(t, CandleOptions{}, b).Doji, 0) != 0 {
		t.Error("not a doji by default")
	}
	if flagAt(t, runCandles(t, CandleOptions{DojiBody: 0.3}, b).Doji, 0) != 1 {
		t.Error("a doji with a looser DojiBody")
	}
}

func TestCandlePatternsRefuseBadInput(t *testing.T) {
	p := prices(10, 11, 12)
	if _, err := CandlePatterns(p, p, p, p, CandleOptions{DojiBody: -1}); err == nil {
		t.Error("negative option")
	}
	if _, err := CandlePatterns(p, p, p, p, CandleOptions{BodyRatio: 2}); err == nil {
		t.Error("a ratio above 1")
	}
	if _, err := CandlePatterns(p, p, p, p, CandleOptions{}, CandleOptions{}); err == nil {
		t.Error("two option sets")
	}
	if _, err := CandlePatterns(prices(10, 11), p, p, p); err == nil {
		t.Error("different lengths")
	}
	if _, err := CandlePatterns(prices(10, 20, 12), p, p, p); err == nil {
		t.Error("an open outside the range")
	}
	if _, err := CandlePatterns(nil, p, p, p); err == nil {
		t.Error("nil open")
	}
	res, err := CandlePatterns(prices(), prices(), prices(), prices())
	if err != nil || res.Doji.Len() != 0 {
		t.Errorf("empty input: %v", err)
	}
}

func TestCandleListsAreNamed(t *testing.T) {
	res := runCandles(t, CandleOptions{}, bar{10, 11, 9, 10})
	for want, l := range map[string]*insyra.DataList{
		"Doji": res.Doji, "Hammer": res.Hammer, "HangingMan": res.HangingMan, "InvertedHammer": res.InvertedHammer,
		"ShootingStar": res.ShootingStar, "Engulfing": res.Engulfing, "Harami": res.Harami, "PiercingLine": res.PiercingLine,
		"DarkCloudCover": res.DarkCloudCover, "MorningStar": res.MorningStar, "EveningStar": res.EveningStar,
		"ThreeWhiteSoldiers": res.ThreeWhiteSoldiers, "ThreeBlackCrows": res.ThreeBlackCrows,
	} {
		if l.GetName() != want {
			t.Errorf("a list is named %q, want %q", l.GetName(), want)
		}
	}
}
