package ta

import (
	"errors"
	"fmt"
	"math"

	"github.com/HazelnutParadise/insyra"
)

// CandleOptions are the optional settings of CandlePatterns. A zero field takes
// the default shown. All the ratios are of the bar's own range (high minus low)
// unless said otherwise.
type CandleOptions struct {
	// DojiBody is the largest body a doji may have. Default 0.1.
	DojiBody float64
	// BodyRatio is the largest body a "small body" may have: the hammers, the
	// stars and the harami use it. Default 0.3.
	BodyRatio float64
	// LongBody is the smallest body a "long" candle has: the first candle of a
	// star or a harami and the engulfed one in piercing lines. Default 0.6.
	LongBody float64
	// LongShadowRatio is how many times the body the long shadow of a hammer or
	// a star must be at least: the shadow is at least body times this. Default 2.
	LongShadowRatio float64
	// ShortShadowRatio is how many times shorter than its reference a shadow
	// that must be small is: the opposite shadow of a hammer or a star is at most
	// the long shadow divided by this, and a shadow of a soldier or a crow is at
	// most the body divided by this. Default 2.
	ShortShadowRatio float64
	// TrendRows is how many rows back the trend before a hammer or a star is
	// read: up when the close of the previous row is above the close TrendRows
	// rows before it. Default 5.
	TrendRows int
	// SegmentStarts are the rows on which a new data segment starts, for example
	// after a trading halt or a relisting. Each segment is computed on its own:
	// no pattern uses a bar of an earlier segment, and the first rows of a segment
	// that lack the bars a pattern needs are 0. Rows are 0-based and must lie
	// within the series; empty gives one segment.
	SegmentStarts []int
}

// CandlePatternResult holds one list per candlestick pattern, as many rows as the
// bars. A flag is on the row whose close completes the pattern, so it is known
// at that close. +1 is bullish, -1 is bearish, and 0 is no pattern; the
// patterns with one direction use only that sign, and Doji is 1 or 0.
type CandlePatternResult struct {
	Doji               *insyra.DataList
	Hammer             *insyra.DataList // +1: small body, long lower shadow, after a fall
	HangingMan         *insyra.DataList // -1: the hammer's shape after a rise
	InvertedHammer     *insyra.DataList // +1: small body, long upper shadow, after a fall
	ShootingStar       *insyra.DataList // -1: the inverted hammer's shape after a rise
	Engulfing          *insyra.DataList // +1 / -1: a body that engulfs the previous opposite body
	Harami             *insyra.DataList // +1 / -1: a small body inside the previous long opposite body
	PiercingLine       *insyra.DataList // +1: opens below the previous low, closes above its body's midpoint
	DarkCloudCover     *insyra.DataList // -1: opens above the previous high, closes below its body's midpoint
	MorningStar        *insyra.DataList // +1: long fall, small gapped body, long rise into the first body
	EveningStar        *insyra.DataList // -1: its mirror
	ThreeWhiteSoldiers *insyra.DataList // +1: three rising bars, each opening inside the last body
	ThreeBlackCrows    *insyra.DataList // -1: three falling bars, each opening inside the last body
}

type candle struct{ o, h, l, c float64 }

func (k candle) body() float64   { return math.Abs(k.c - k.o) }
func (k candle) rng() float64    { return k.h - k.l }
func (k candle) top() float64    { return math.Max(k.o, k.c) }
func (k candle) bottom() float64 { return math.Min(k.o, k.c) }
func (k candle) upper() float64  { return k.h - k.top() }
func (k candle) lower() float64  { return k.bottom() - k.l }
func (k candle) up() bool        { return k.c > k.o }
func (k candle) down() bool      { return k.c < k.o }
func (k candle) mid() float64    { return (k.o + k.c) / 2 }

// CandlePatterns flags the common candlestick patterns in a series of daily bars
// with ratios set in CandleOptions. The definitions are the plain textbook ones;
// platforms differ on the cut-offs, which is why they are options, and TA-Lib's
// candle functions use different rules and will not match.
func CandlePatterns(open, high, low, close insyra.IDataList, opts ...CandleOptions) (*CandlePatternResult, error) {
	if len(opts) > 1 {
		return nil, errors.New("CandlePatterns accepts at most one CandleOptions")
	}
	var o CandleOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	if o.DojiBody < 0 || o.BodyRatio < 0 || o.LongBody < 0 || o.LongShadowRatio < 0 || o.ShortShadowRatio < 0 || o.TrendRows < 0 {
		return nil, errors.New("CandleOptions fields must not be negative")
	}
	for name, v := range map[string]float64{"DojiBody": o.DojiBody, "BodyRatio": o.BodyRatio, "LongBody": o.LongBody} {
		if v > 1 {
			return nil, fmt.Errorf("%s is a share of the range and must not exceed 1, got %v", name, v)
		}
	}
	if o.DojiBody == 0 {
		o.DojiBody = 0.1
	}
	if o.BodyRatio == 0 {
		o.BodyRatio = 0.3
	}
	if o.LongBody == 0 {
		o.LongBody = 0.6
	}
	if o.LongShadowRatio == 0 {
		o.LongShadowRatio = 2
	}
	if o.ShortShadowRatio == 0 {
		o.ShortShadowRatio = 2
	}
	if o.TrendRows == 0 {
		o.TrendRows = 5
	}
	op, err := numericSeries(open, "open")
	if err != nil {
		return nil, err
	}
	h, l, c, err := barSeries(high, low, close)
	if err != nil {
		return nil, err
	}
	if len(op) != len(c) {
		return nil, fmt.Errorf("open has %d rows but close has %d", len(op), len(c))
	}
	n := len(c)
	starts, err := cleanSegmentStarts(o.SegmentStarts, n)
	if err != nil {
		return nil, err
	}
	bars := make([]candle, n)
	for i := range bars {
		if op[i] > h[i] || op[i] < l[i] || c[i] > h[i] || c[i] < l[i] {
			return nil, fmt.Errorf("the open %v or close %v lies outside the range %v to %v at row index %d", op[i], c[i], l[i], h[i], i)
		}
		bars[i] = candle{op[i], h[i], l[i], c[i]}
	}
	names := []string{"Doji", "Hammer", "HangingMan", "InvertedHammer", "ShootingStar", "Engulfing", "Harami", "PiercingLine", "DarkCloudCover", "MorningStar", "EveningStar", "ThreeWhiteSoldiers", "ThreeBlackCrows"}
	flags := make(map[string][]any, len(names))
	for _, name := range names {
		flags[name] = make([]any, n)
		for i := range flags[name] {
			flags[name][i] = 0
		}
	}
	for _, seg := range segmentBounds(n, starts) {
		candleSegment(bars[seg[0]:seg[1]], o, flags, seg[0])
	}
	list := func(name string) *insyra.DataList { return insyra.NewDataList(flags[name]...).SetName(name) }
	return &CandlePatternResult{
		Doji: list("Doji"), Hammer: list("Hammer"), HangingMan: list("HangingMan"), InvertedHammer: list("InvertedHammer"),
		ShootingStar: list("ShootingStar"), Engulfing: list("Engulfing"), Harami: list("Harami"), PiercingLine: list("PiercingLine"),
		DarkCloudCover: list("DarkCloudCover"), MorningStar: list("MorningStar"), EveningStar: list("EveningStar"),
		ThreeWhiteSoldiers: list("ThreeWhiteSoldiers"), ThreeBlackCrows: list("ThreeBlackCrows"),
	}, nil
}

// candleSegment flags the patterns of one segment of bars into flags, whose row
// off is the segment's first row. It reads nothing outside bars.
func candleSegment(bars []candle, o CandleOptions, flags map[string][]any, off int) {
	c := make([]float64, len(bars))
	for i, k := range bars {
		c[i] = k.c
	}
	small := func(k candle) bool { return k.rng() > 0 && k.body() <= o.BodyRatio*k.rng() }
	long := func(k candle) bool { return k.rng() > 0 && k.body() >= o.LongBody*k.rng() }
	lowerShadowed := func(k candle) bool {
		return small(k) && k.lower() > 0 && k.lower() >= o.LongShadowRatio*k.body() && k.upper() <= k.lower()/o.ShortShadowRatio
	}
	upperShadowed := func(k candle) bool {
		return small(k) && k.upper() > 0 && k.upper() >= o.LongShadowRatio*k.body() && k.lower() <= k.upper()/o.ShortShadowRatio
	}
	trend := func(i int) int { // before row i: 1 up, -1 down, 0 unknown
		if i-1-o.TrendRows < 0 {
			return 0
		}
		switch {
		case c[i-1] > c[i-1-o.TrendRows]:
			return 1
		case c[i-1] < c[i-1-o.TrendRows]:
			return -1
		}
		return 0
	}
	for i, k := range bars {
		if k.rng() > 0 && k.body() <= o.DojiBody*k.rng() {
			flags["Doji"][off+i] = 1
		}
		if lowerShadowed(k) {
			switch trend(i) {
			case -1:
				flags["Hammer"][off+i] = 1
			case 1:
				flags["HangingMan"][off+i] = -1
			}
		}
		if upperShadowed(k) {
			switch trend(i) {
			case -1:
				flags["InvertedHammer"][off+i] = 1
			case 1:
				flags["ShootingStar"][off+i] = -1
			}
		}
		if i >= 1 {
			p := bars[i-1]
			switch {
			case p.down() && k.up() && k.o <= p.c && k.c >= p.o && k.body() > p.body():
				flags["Engulfing"][off+i] = 1
			case p.up() && k.down() && k.o >= p.c && k.c <= p.o && k.body() > p.body():
				flags["Engulfing"][off+i] = -1
			}
			if long(p) && k.body() < p.body() && k.top() <= p.top() && k.bottom() >= p.bottom() && k.body() > 0 {
				switch {
				case p.down() && k.up():
					flags["Harami"][off+i] = 1
				case p.up() && k.down():
					flags["Harami"][off+i] = -1
				}
			}
			if long(p) && p.down() && k.up() && k.o < p.l && k.c > p.mid() && k.c < p.o {
				flags["PiercingLine"][off+i] = 1
			}
			if long(p) && p.up() && k.down() && k.o > p.h && k.c < p.mid() && k.c > p.o {
				flags["DarkCloudCover"][off+i] = -1
			}
		}
		if i >= 2 {
			a, b := bars[i-2], bars[i-1]
			if long(a) && a.down() && small(b) && b.top() < a.c && k.up() && k.c > a.mid() {
				flags["MorningStar"][off+i] = 1
			}
			if long(a) && a.up() && small(b) && b.bottom() > a.c && k.down() && k.c < a.mid() {
				flags["EveningStar"][off+i] = -1
			}
			noUpperWick := func(x candle) bool { return x.upper()*o.ShortShadowRatio <= x.body() }
			noLowerWick := func(x candle) bool { return x.lower()*o.ShortShadowRatio <= x.body() }
			if a.up() && b.up() && k.up() && b.c > a.c && k.c > b.c &&
				b.o > a.o && b.o < a.c && k.o > b.o && k.o < b.c && noUpperWick(a) && noUpperWick(b) && noUpperWick(k) {
				flags["ThreeWhiteSoldiers"][off+i] = 1
			}
			if a.down() && b.down() && k.down() && b.c < a.c && k.c < b.c &&
				b.o < a.o && b.o > a.c && k.o < b.o && k.o > b.c && noLowerWick(a) && noLowerWick(b) && noLowerWick(k) {
				flags["ThreeBlackCrows"][off+i] = -1
			}
		}
	}
}
