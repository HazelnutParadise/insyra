package ta

import (
	"encoding/json"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

type indicatorCase struct {
	High, Low, Close []float64
	Expected         map[string]json.RawMessage
}

func indicatorCases(t *testing.T) []indicatorCase {
	t.Helper()
	raw, err := os.ReadFile("testdata/indicator_baseline.json")
	if err != nil {
		t.Fatal(err)
	}
	var f struct{ Cases []indicatorCase }
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	return f.Cases
}

func (c indicatorCase) series(t *testing.T, key string) []*float64 {
	t.Helper()
	var out []*float64
	if err := json.Unmarshal(c.Expected[key], &out); err != nil {
		t.Fatalf("%s: %v", key, err)
	}
	return out
}

func (c indicatorCase) lines(t *testing.T, key string) map[string][]*float64 {
	t.Helper()
	var out map[string][]*float64
	if err := json.Unmarshal(c.Expected[key], &out); err != nil {
		t.Fatalf("%s: %v", key, err)
	}
	return out
}

// sameAsReference checks got against a reference series in which null is NaN.
func sameAsReference(t *testing.T, name string, got *insyra.DataList, want []*float64) {
	t.Helper()
	g := floats(t, got)
	if len(g) != len(want) {
		t.Fatalf("%s: %d rows, reference has %d", name, len(g), len(want))
	}
	checked := 0
	for i, w := range want {
		if w == nil {
			if !math.IsNaN(g[i]) {
				t.Fatalf("%s row %d: got %v, reference is NaN", name, i, g[i])
			}
			continue
		}
		if math.IsNaN(g[i]) || math.Abs(g[i]-*w) > 1e-9*math.Max(1, math.Abs(*w)) {
			t.Fatalf("%s row %d: got %v, reference %v", name, i, g[i], *w)
		}
		checked++
	}
	if checked == 0 {
		t.Fatalf("%s: nothing to compare", name)
	}
}

func TestIndicatorsMatchTALib(t *testing.T) {
	for ci, c := range indicatorCases(t) {
		cl, hi, lo := prices(c.Close...), prices(c.High...), prices(c.Low...)
		name := func(s string) string { return s + " case " + string(rune('0'+ci)) }

		for _, p := range []int{1, 5, 20} {
			got, err := SMA(cl, p)
			if err != nil {
				t.Fatal(err)
			}
			sameAsReference(t, name("sma"), got, c.series(t, "sma"+itoa(p)))
		}
		for _, p := range []int{3, 12, 26} {
			got, err := EMA(cl, p)
			if err != nil {
				t.Fatal(err)
			}
			sameAsReference(t, name("ema"), got, c.series(t, "ema"+itoa(p)))
		}
		for _, s := range [][3]int{{12, 26, 9}, {5, 13, 4}, {3, 8, 1}} {
			got, err := MACD(cl, s[0], s[1], s[2])
			if err != nil {
				t.Fatal(err)
			}
			want := c.lines(t, "macd"+itoa(s[0])+"_"+itoa(s[1])+"_"+itoa(s[2]))
			// TA-Lib leaves the MACD line empty until the signal line exists; here it starts with the slow EMA.
			sameWhereDefined(t, name("macd"), got.MACD, want["macd"], s[1]-1)
			sameAsReference(t, name("macd signal"), got.Signal, want["signal"])
			sameAsReference(t, name("macd hist"), got.Histogram, want["hist"])
		}
		for _, b := range []struct {
			n   int
			k   float64
			key string
		}{{20, 2, "boll20_2.0"}, {10, 1.5, "boll10_1.5"}, {5, 0, "boll5_0.0"}} {
			got, err := Bollinger(cl, b.n, b.k)
			if err != nil {
				t.Fatal(err)
			}
			want := c.lines(t, b.key)
			sameAsReference(t, name("upper"), got.Upper, want["upper"])
			sameAsReference(t, name("middle"), got.Middle, want["middle"])
			sameAsReference(t, name("lower"), got.Lower, want["lower"])
		}
		tr, err := TrueRange(hi, lo, cl)
		if err != nil {
			t.Fatal(err)
		}
		sameAsReference(t, name("trange"), tr, c.series(t, "trange"))
		for _, p := range []int{1, 5, 14} {
			got, err := ATR(hi, lo, cl, p)
			if err != nil {
				t.Fatal(err)
			}
			sameAsReference(t, name("atr"), got, c.series(t, "atr"+itoa(p)))
		}
		for _, s := range [][3]int{{9, 3, 3}, {5, 2, 4}, {14, 1, 1}} {
			key := itoa(s[0]) + "_" + itoa(s[1]) + "_" + itoa(s[2])
			opts := KDOptions{SlowK: s[1], SlowD: s[2]}
			got, err := KD(hi, lo, cl, s[0], opts)
			if err != nil {
				t.Fatal(err)
			}
			want := c.lines(t, "kd"+key)
			sameAsReference(t, name("kd k"), got.K, want["k"])
			sameAsReference(t, name("kd d"), got.D, want["d"])

			opts.Method = KDMovingAverage
			got, err = KD(hi, lo, cl, s[0], opts)
			if err != nil {
				t.Fatal(err)
			}
			want = c.lines(t, "stoch"+key)
			// As with MACD, TA-Lib starts K together with D.
			sameWhereDefined(t, name("stoch k"), got.K, want["k"], s[0]+s[1]-2)
			sameAsReference(t, name("stoch d"), got.D, want["d"])
		}
	}
}

// sameWhereDefined compares only the rows the reference has, and requires got to be defined from row first.
func sameWhereDefined(t *testing.T, name string, got *insyra.DataList, want []*float64, first int) {
	t.Helper()
	g := floats(t, got)
	for i := range g {
		if (i < first) != math.IsNaN(g[i]) {
			t.Fatalf("%s row %d: got %v, the line should start at row %d", name, i, g[i], first)
		}
		if want[i] != nil && math.Abs(g[i]-*want[i]) > 1e-9*math.Max(1, math.Abs(*want[i])) {
			t.Fatalf("%s row %d: got %v, reference %v", name, i, g[i], *want[i])
		}
	}
}

func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return itoa(n/10) + string(rune('0'+n%10))
}

func TestIndicatorsByHand(t *testing.T) {
	sma, _ := SMA(prices(1, 2, 3, 4, 5), 3)
	nan := math.NaN()
	want := []float64{nan, nan, 2, 3, 4}
	g := floats(t, sma)
	for i := range want {
		if !(math.IsNaN(want[i]) && math.IsNaN(g[i])) && g[i] != want[i] {
			t.Fatalf("sma: %v", g)
		}
	}
	ema, _ := EMA(prices(1, 2, 3, 4, 5), 3) // seed 2, weight 0.5
	g = floats(t, ema)
	if g[2] != 2 || g[3] != 3 || g[4] != 4 {
		t.Fatalf("ema: %v", g)
	}
	// Close at the top of a 3-row range: RSV 100. Recursive K from 50: 50*2/3 + 100/3.
	kd, err := KD(prices(2, 3, 4), prices(1, 2, 3), prices(2, 3, 4), 3)
	if err != nil {
		t.Fatal(err)
	}
	k, d := floats(t, kd.K), floats(t, kd.D)
	wantK := 50.0*2/3 + 100.0/3
	if math.Abs(k[2]-wantK) > 1e-12 || math.Abs(d[2]-(50.0*2/3+wantK/3)) > 1e-12 || !math.IsNaN(k[1]) {
		t.Fatalf("kd: %v %v", k, d)
	}
	// A flat window has no range and counts as 50.
	flat, _ := KD(prices(5, 5, 5), prices(5, 5, 5), prices(5, 5, 5), 2)
	if f := floats(t, flat.K); f[1] != 50 || f[2] != 50 {
		t.Fatalf("flat window: %v", f)
	}
}

func TestIndicatorsRefuseBadInput(t *testing.T) {
	p := prices(1, 2, 3, 4, 5, 6)
	if _, err := SMA(p, 0); err == nil {
		t.Error("n 0")
	}
	if _, err := EMA(p, 0); err == nil {
		t.Error("n 0")
	}
	if _, err := MACD(p, 5, 3, 2); err == nil {
		t.Error("slow not above fast")
	}
	if _, err := MACD(p, 1, 3, 2); err == nil {
		t.Error("fast below 2")
	}
	if _, err := Bollinger(p, 1, 2); err == nil {
		t.Error("n 1")
	}
	if _, err := Bollinger(p, 3, -1); err == nil {
		t.Error("negative k")
	}
	if _, err := Bollinger(p, 3, math.NaN()); err == nil {
		t.Error("NaN k")
	}
	if _, err := ATR(p, prices(1, 1, 1, 1, 1), p, 2); err == nil {
		t.Error("different lengths")
	}
	if _, err := ATR(prices(1, 1), prices(2, 2), prices(1, 1), 1); err == nil {
		t.Error("high below low")
	}
	if _, err := KD(p, p, p, 3, KDOptions{Method: KDMethod(9)}); err == nil || !strings.Contains(err.Error(), "KDMethod(9)") {
		t.Errorf("unknown method: %v", err)
	}
	if _, err := KD(p, p, p, 3, KDOptions{SlowK: -1}); err == nil {
		t.Error("negative SlowK")
	}
	if _, err := KD(p, p, p, 3, KDOptions{}, KDOptions{}); err == nil {
		t.Error("two options")
	}
	if _, err := SMA(prices(1, math.NaN(), 3), 2); err == nil || !strings.Contains(err.Error(), "row index 1") {
		t.Errorf("a NaN price should be refused with its row: %v", err)
	}
	if _, err := SMA(nil, 2); err == nil {
		t.Error("nil prices")
	}
	// Fewer rows than the period: all NaN, not an error.
	short, err := ATR(prices(2, 3), prices(1, 2), prices(1.5, 2.5), 5)
	if err != nil || len(floats(t, short)) != 2 || !math.IsNaN(floats(t, short)[1]) {
		t.Errorf("short series: %v", err)
	}
}

func TestIndicatorListsAreNamed(t *testing.T) {
	p := prices(1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12)
	one := func(l *insyra.DataList, err error, want string) {
		t.Helper()
		if err != nil || l.GetName() != want {
			t.Errorf("want a list named %q, got %q, %v", want, l.GetName(), err)
		}
	}
	l, err := SMA(p, 3)
	one(l, err, "SMA")
	l, err = EMA(p, 3)
	one(l, err, "EMA")
	m, err := MACD(p, 2, 4, 2)
	one(m.MACD, err, "MACD")
	one(m.Signal, err, "Signal")
	one(m.Histogram, err, "Histogram")
	b, err := Bollinger(p, 3, 2)
	one(b.Upper, err, "Upper")
	one(b.Middle, err, "Middle")
	one(b.Lower, err, "Lower")
	l, err = TrueRange(p, p, p)
	one(l, err, "TrueRange")
	l, err = ATR(p, p, p, 3)
	one(l, err, "ATR")
	k, err := KD(p, p, p, 3)
	one(k.K, err, "K")
	one(k.D, err, "D")
}
