package ta

import (
	"encoding/json"
	"os"
	"testing"
)

func trendCases(t *testing.T) []indicatorCase {
	t.Helper()
	raw, err := os.ReadFile("testdata/trend_baseline.json")
	if err != nil {
		t.Fatal(err)
	}
	var f struct{ Cases []indicatorCase }
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	return f.Cases
}

func TestTrendIndicatorsMatchTheReference(t *testing.T) {
	for ci, c := range trendCases(t) {
		cl, hi, lo := prices(c.Close...), prices(c.High...), prices(c.Low...)
		name := func(s string) string { return s + " case " + itoa(ci) }
		for _, n := range []int{14, 5} {
			got, err := Aroon(hi, lo, n)
			if err != nil {
				t.Fatal(err)
			}
			want := c.lines(t, "aroon"+itoa(n))
			sameAsReference(t, name("aroon up"), got.Up, want["up"])
			sameAsReference(t, name("aroon down"), got.Down, want["down"])
			sameAsReference(t, name("aroon osc"), got.Oscillator, want["osc"])
		}
		for _, s := range [][2]int{{15, 9}, {5, 3}} {
			got, err := TRIX(cl, s[0], s[1])
			if err != nil {
				t.Fatal(err)
			}
			want := c.lines(t, "trix"+itoa(s[0])+"_"+itoa(s[1]))
			sameAsReference(t, name("trix"), got.TRIX, want["trix"])
			sameAsReference(t, name("trix signal"), got.Signal, want["signal"])
		}
		for _, s := range []struct {
			n, fast, slow int
			key           string
		}{{10, 2, 30, "kama10_2_30"}, {30, 2, 30, "kama30_2_30"}, {6, 3, 20, "kama6_3_20"}} {
			got, err := KAMA(cl, s.n, s.fast, s.slow)
			if err != nil {
				t.Fatal(err)
			}
			sameWhereDefined(t, name(s.key), got, c.series(t, s.key), s.n)
		}
		for _, s := range []struct {
			tk, kj, sb, d int
			key           string
		}{{9, 26, 52, 26, "ichimoku9_26_52_26"}, {5, 10, 20, 10, "ichimoku5_10_20_10"}} {
			got, err := Ichimoku(hi, lo, cl, s.tk, s.kj, s.sb, IchimokuOptions{Displacement: s.d})
			if err != nil {
				t.Fatal(err)
			}
			want := c.lines(t, s.key)
			sameAsReference(t, name(s.key+" tenkan"), got.Tenkan, want["tenkan"])
			sameAsReference(t, name(s.key+" kijun"), got.Kijun, want["kijun"])
			sameAsReference(t, name(s.key+" senkouA"), got.SenkouA, want["senkouA"])
			sameAsReference(t, name(s.key+" senkouB"), got.SenkouB, want["senkouB"])
		}
	}
}

func TestIchimokuCloudUsesOnlyEarlierRows(t *testing.T) {
	n := 120
	hs, ls, cs := make([]float64, n), make([]float64, n), make([]float64, n)
	for i := range hs {
		cs[i] = 100 + float64(i%17) + float64(i)/10
		hs[i], ls[i] = cs[i]+1, cs[i]-1
	}
	full, err := Ichimoku(prices(hs...), prices(ls...), prices(cs...), 9, 26, 52)
	if err != nil {
		t.Fatal(err)
	}
	a := floats(t, full.SenkouA)
	// The cloud on row t does not change when later rows do.
	hs2 := append([]float64(nil), hs...)
	ls2 := append([]float64(nil), ls...)
	cs2 := append([]float64(nil), cs...)
	for i := 60; i < n; i++ {
		hs2[i], ls2[i], cs2[i] = hs[i]*2, ls[i]*2, cs[i]*2
	}
	changed, _ := Ichimoku(prices(hs2...), prices(ls2...), prices(cs2...), 9, 26, 52)
	b := floats(t, changed.SenkouA)
	for i := 0; i < 60+26; i++ {
		if !(a[i] == b[i] || (a[i] != a[i] && b[i] != b[i])) {
			t.Fatalf("row %d changed when only rows from 60 on did", i)
		}
	}
	// Row 26+51 is the first row senkou B exists (52 rows of history, then drawn 26 ahead).
	sb := floats(t, full.SenkouB)
	if sb[26+50] == sb[26+50] || sb[26+51] != sb[26+51] {
		t.Fatalf("senkou B first value at the wrong row: %v", sb[70:80])
	}
}

func TestTrendRefuseBadInput(t *testing.T) {
	p := prices(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
	if _, err := Aroon(p, p, 0); err == nil {
		t.Error("aroon n 0")
	}
	if _, err := Aroon(prices(1, 2), prices(1), 1); err == nil {
		t.Error("aroon lengths")
	}
	if _, err := TRIX(p, 0, 3); err == nil {
		t.Error("trix n 0")
	}
	if _, err := TRIX(p, 3, 0); err == nil {
		t.Error("trix signal 0")
	}
	if _, err := KAMA(p, 3, 5, 5); err == nil {
		t.Error("kama slow not above fast")
	}
	if _, err := KAMA(p, 0, 2, 30); err == nil {
		t.Error("kama n 0")
	}
	if _, err := Ichimoku(p, p, p, 0, 26, 52); err == nil {
		t.Error("ichimoku tenkan 0")
	}
	if _, err := Ichimoku(p, p, p, 9, 26, 52, IchimokuOptions{Displacement: -1}); err == nil {
		t.Error("negative displacement")
	}
	if _, err := Ichimoku(p, p, p, 9, 26, 52, IchimokuOptions{}, IchimokuOptions{}); err == nil {
		t.Error("two option sets")
	}
	if _, err := KAMA(nil, 3, 2, 30); err == nil {
		t.Error("nil prices")
	}
	short, err := KAMA(prices(1, 2), 5, 2, 30)
	if err != nil || len(floats(t, short)) != 2 {
		t.Errorf("fewer rows than n: %v", err)
	}
}

func TestTrendListsAreNamed(t *testing.T) {
	p := prices(1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20)
	a, _ := Aroon(p, p, 3)
	x, _ := TRIX(p, 2, 2)
	k, _ := KAMA(p, 3, 2, 30)
	i, _ := Ichimoku(p, p, p, 2, 3, 4, IchimokuOptions{Displacement: 2})
	for want, got := range map[string]string{
		"Up": a.Up.GetName(), "Down": a.Down.GetName(), "Oscillator": a.Oscillator.GetName(),
		"TRIX": x.TRIX.GetName(), "Signal": x.Signal.GetName(), "KAMA": k.GetName(),
		"Tenkan": i.Tenkan.GetName(), "Kijun": i.Kijun.GetName(), "SenkouA": i.SenkouA.GetName(), "SenkouB": i.SenkouB.GetName(),
	} {
		if got != want {
			t.Errorf("a list is named %q, want %q", got, want)
		}
	}
}
