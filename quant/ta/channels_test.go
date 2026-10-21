package ta

import (
	"encoding/json"
	"math"
	"os"
	"testing"
)

func channelCases(t *testing.T) []indicatorCase {
	t.Helper()
	raw, err := os.ReadFile("testdata/channels_baseline.json")
	if err != nil {
		t.Fatal(err)
	}
	var f struct{ Cases []indicatorCase }
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	return f.Cases
}

func TestChannelsAndStopsMatchTheReference(t *testing.T) {
	for ci, c := range channelCases(t) {
		cl, hi, lo := prices(c.Close...), prices(c.High...), prices(c.Low...)
		name := func(s string) string { return s + " case " + itoa(ci) }
		for _, s := range []struct {
			n, atrN int
			k       float64
			key     string
		}{{20, 10, 2, "keltner20_10_2"}, {10, 5, 1.5, "keltner10_5_1.5"}} {
			got, err := Keltner(hi, lo, cl, s.n, s.atrN, s.k)
			if err != nil {
				t.Fatal(err)
			}
			want := c.lines(t, s.key)
			sameAsReference(t, name(s.key+" upper"), got.Upper, want["upper"])
			sameAsReference(t, name(s.key+" middle"), got.Middle, want["middle"])
			sameAsReference(t, name(s.key+" lower"), got.Lower, want["lower"])
		}
		for _, n := range []int{20, 5} {
			got, err := KeltnerClassic(hi, lo, cl, n)
			if err != nil {
				t.Fatal(err)
			}
			want := c.lines(t, "keltnerClassic"+itoa(n))
			sameAsReference(t, name("classic upper"), got.Upper, want["upper"])
			sameAsReference(t, name("classic middle"), got.Middle, want["middle"])
			sameAsReference(t, name("classic lower"), got.Lower, want["lower"])
			d, err := Donchian(hi, lo, n)
			if err != nil {
				t.Fatal(err)
			}
			want = c.lines(t, "donchian"+itoa(n))
			sameAsReference(t, name("donchian upper"), d.Upper, want["upper"])
			sameAsReference(t, name("donchian middle"), d.Middle, want["middle"])
			sameAsReference(t, name("donchian lower"), d.Lower, want["lower"])
		}
		for _, s := range []struct {
			n   int
			k   float64
			key string
		}{{10, 3, "supertrend10_3.0"}, {7, 2, "supertrend7_2.0"}} {
			got, err := SuperTrend(hi, lo, cl, s.n, s.k)
			if err != nil {
				t.Fatal(err)
			}
			want := c.lines(t, s.key)
			sameAsReference(t, name(s.key+" line"), got.Line, want["line"])
			sameAsReference(t, name(s.key+" direction"), got.Direction, want["direction"])
		}
		for _, s := range []struct {
			n   int
			k   float64
			key string
		}{{22, 3, "chandelier22_3.0"}, {10, 2, "chandelier10_2.0"}} {
			got, err := Chandelier(hi, lo, cl, s.n, s.k)
			if err != nil {
				t.Fatal(err)
			}
			want := c.lines(t, s.key)
			sameAsReference(t, name(s.key+" long"), got.Long, want["long"])
			sameAsReference(t, name(s.key+" short"), got.Short, want["short"])
		}
		for _, s := range []struct {
			start, step, max float64
			key              string
		}{{0.02, 0.02, 0.2, "sar0.02_0.02_0.2"}, {0.01, 0.03, 0.15, "sar0.01_0.03_0.15"}} {
			got, err := ParabolicSAR(hi, lo, s.start, s.step, s.max)
			if err != nil {
				t.Fatal(err)
			}
			sameAsReference(t, name(s.key), got, c.series(t, s.key))
		}
	}
}

func TestChannelsByHand(t *testing.T) {
	h := prices(10, 12, 11, 13, 12)
	l := prices(8, 9, 9, 10, 10)
	d, err := Donchian(h, l, 2)
	if err != nil {
		t.Fatal(err)
	}
	// Row 2 looks at rows 0 and 1: highest high 12, lowest low 8. The current row is not in it.
	up, lo := floats(t, d.Upper), floats(t, d.Lower)
	if !(up[0] != up[0]) || up[2] != 12 || lo[2] != 8 || up[3] != 12 || lo[3] != 9 || up[4] != 13 || lo[4] != 9 {
		t.Fatalf("upper %v lower %v", up, lo)
	}
	// A flat series has zero ATR, so the chandelier lines sit on the extremes and
	// the SuperTrend line sits on the price.
	flat := prices(5, 5, 5, 5, 5, 5)
	ch, err := Chandelier(flat, flat, flat, 2, 3)
	if err != nil {
		t.Fatal(err)
	}
	if lg := floats(t, ch.Long); lg[2] != 5 || lg[5] != 5 {
		t.Fatalf("long %v", lg)
	}
	st, err := SuperTrend(flat, flat, flat, 2, 3)
	if err != nil {
		t.Fatal(err)
	}
	if line := floats(t, st.Line); line[2] != 5 || line[5] != 5 {
		t.Fatalf("line %v", line)
	}
}

func TestChannelsRefuseBadInput(t *testing.T) {
	p := prices(10, 11, 12, 13, 14, 15)
	if _, err := Keltner(p, p, p, 0, 10, 2); err == nil {
		t.Error("n 0")
	}
	if _, err := Keltner(p, p, p, 3, 0, 2); err == nil {
		t.Error("atrN 0")
	}
	if _, err := Keltner(p, p, p, 3, 3, -1); err == nil {
		t.Error("negative k")
	}
	if _, err := KeltnerClassic(p, p, p, 0); err == nil {
		t.Error("classic n 0")
	}
	if _, err := Donchian(p, prices(1, 2, 3), 2); err == nil {
		t.Error("different lengths")
	}
	if _, err := Donchian(prices(1, 2, 3), prices(2, 3, 4), 2); err == nil {
		t.Error("high below low")
	}
	if _, err := SuperTrend(p, p, p, 3, 0); err == nil {
		t.Error("k 0")
	}
	if _, err := ParabolicSAR(p, p, 0, 0.02, 0.2); err == nil {
		t.Error("start 0")
	}
	if _, err := ParabolicSAR(p, p, 0.1, 0.02, 0.05); err == nil {
		t.Error("max below start")
	}
	if _, err := Chandelier(p, p, p, 3, -1); err == nil {
		t.Error("negative k")
	}
	if _, err := Donchian(nil, p, 2); err == nil {
		t.Error("nil high")
	}
	if short, err := ParabolicSAR(prices(1), prices(1), 0.02, 0.02, 0.2); err != nil || len(floats(t, short)) != 1 {
		t.Errorf("one row: %v", err)
	}
}

func TestChannelListsAreNamed(t *testing.T) {
	p := prices(1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12)
	k, _ := Keltner(p, p, p, 3, 3, 2)
	d, _ := Donchian(p, p, 3)
	s, _ := SuperTrend(p, p, p, 3, 2)
	c, _ := Chandelier(p, p, p, 3, 2)
	sar, _ := ParabolicSAR(p, p, 0.02, 0.02, 0.2)
	for want, got := range map[string]string{
		"Upper": k.Upper.GetName(), "Middle": d.Middle.GetName(), "Lower": d.Lower.GetName(),
		"SuperTrend": s.Line.GetName(), "Direction": s.Direction.GetName(),
		"Long": c.Long.GetName(), "Short": c.Short.GetName(), "SAR": sar.GetName(),
	} {
		if got != want {
			t.Errorf("a list is named %q, want %q", got, want)
		}
	}
}

func TestTheMultiplierMustBeAboveZeroInEveryChannel(t *testing.T) {
	p := prices(10, 11, 12, 13, 14, 15)
	for _, k := range []float64{0, -1, math.NaN(), math.Inf(1), math.Inf(-1)} {
		if _, err := Keltner(p, p, p, 3, 3, k); err == nil {
			t.Errorf("Keltner accepted k %v", k)
		}
		if _, err := SuperTrend(p, p, p, 3, k); err == nil {
			t.Errorf("SuperTrend accepted k %v", k)
		}
		if _, err := Chandelier(p, p, p, 3, k); err == nil {
			t.Errorf("Chandelier accepted k %v", k)
		}
	}
	_, e1 := Keltner(p, p, p, 3, 3, 0)
	_, e2 := SuperTrend(p, p, p, 3, 0)
	_, e3 := Chandelier(p, p, p, 3, 0)
	if e1 == nil || e2 == nil || e3 == nil || e1.Error() != e2.Error() || e2.Error() != e3.Error() {
		t.Errorf("the three errors should read the same: %v / %v / %v", e1, e2, e3)
	}
}
