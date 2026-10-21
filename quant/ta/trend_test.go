package ta

import (
	"encoding/json"
	"math"
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
		"Chikou": i.Chikou.GetName(), "Tenkan": i.Tenkan.GetName(), "Kijun": i.Kijun.GetName(), "SenkouA": i.SenkouA.GetName(), "SenkouB": i.SenkouB.GetName(),
	} {
		if got != want {
			t.Errorf("a list is named %q, want %q", got, want)
		}
	}
}

func ichimokuBars(n int) (hs, ls, cs []float64) {
	hs, ls, cs = make([]float64, n), make([]float64, n), make([]float64, n)
	for i := range hs {
		cs[i] = 100 + float64(i%13)*1.5 + float64(i)/7
		hs[i], ls[i] = cs[i]+1+float64(i%3), cs[i]-1-float64(i%5)
	}
	return
}

func ichimokuLines(t *testing.T, r *IchimokuResult) [][]float64 {
	t.Helper()
	return [][]float64{floats(t, r.Tenkan), floats(t, r.Kijun), floats(t, r.SenkouA), floats(t, r.SenkouB), floats(t, r.Chikou)}
}

func TestIchimokuChikouIsTheCloseDrawnBack(t *testing.T) {
	hs, ls, cs := ichimokuBars(80)
	for _, d := range []int{26, 5} {
		r, err := Ichimoku(prices(hs...), prices(ls...), prices(cs...), 9, 26, 52, IchimokuOptions{Displacement: d})
		if err != nil {
			t.Fatal(err)
		}
		ch := floats(t, r.Chikou)
		if len(ch) != len(cs) || r.Chikou.GetName() != "Chikou" {
			t.Fatalf("chikou has %d rows named %q", len(ch), r.Chikou.GetName())
		}
		for i := range ch {
			if i+d < len(cs) {
				if ch[i] != cs[i+d] {
					t.Fatalf("d %d row %d: chikou %v, want close[%d] %v", d, i, ch[i], i+d, cs[i+d])
				}
			} else if ch[i] == ch[i] {
				t.Fatalf("d %d row %d: chikou %v, want NaN in the last %d rows", d, i, ch[i], d)
			}
		}
	}
}

func TestIchimokuDefaultDisplacementShiftsChikou26Rows(t *testing.T) {
	hs, ls, cs := ichimokuBars(40)
	r, err := Ichimoku(prices(hs...), prices(ls...), prices(cs...), 9, 26, 52)
	if err != nil {
		t.Fatal(err)
	}
	ch := floats(t, r.Chikou)
	if ch[13] != cs[39] || ch[14] == ch[14] {
		t.Fatalf("default chikou: row 13 %v (want %v), row 14 %v (want NaN)", ch[13], cs[39], ch[14])
	}
}

func TestIchimokuSegmentsAreComputedOnTheirOwn(t *testing.T) {
	hs, ls, cs := ichimokuBars(130)
	for _, starts := range [][]int{{60}, {60, 60, 0}, {30, 90}, {125}} {
		for _, d := range []int{26, 4} {
			o := IchimokuOptions{Displacement: d, SegmentStarts: starts}
			got, err := Ichimoku(prices(hs...), prices(ls...), prices(cs...), 5, 10, 20, o)
			if err != nil {
				t.Fatal(err)
			}
			g := ichimokuLines(t, got)
			clean, _ := cleanSegmentStarts(starts, len(cs))
			for _, b := range segmentBounds(len(cs), clean) {
				part, err := Ichimoku(prices(hs[b[0]:b[1]]...), prices(ls[b[0]:b[1]]...), prices(cs[b[0]:b[1]]...), 5, 10, 20, IchimokuOptions{Displacement: d})
				if err != nil {
					t.Fatal(err)
				}
				for li, want := range ichimokuLines(t, part) {
					if !sameFloats(g[li][b[0]:b[1]], want) {
						t.Fatalf("starts %v d %d line %d segment %v differs from computing the segment alone", starts, d, li, b)
					}
				}
			}
		}
	}
}

func TestIchimokuNothingCrossesASegmentBoundary(t *testing.T) {
	hs, ls, cs := ichimokuBars(100)
	d := 10
	r, err := Ichimoku(prices(hs...), prices(ls...), prices(cs...), 5, 10, 20, IchimokuOptions{Displacement: d, SegmentStarts: []int{50}})
	if err != nil {
		t.Fatal(err)
	}
	g := ichimokuLines(t, r)
	for _, row := range []int{50, 50 + d - 1} { // the cloud of the new segment has nothing yet
		if g[2][row] == g[2][row] || g[3][row] == g[3][row] {
			t.Fatalf("cloud on row %d has a value borrowed from the segment before", row)
		}
	}
	for row := 50 - d; row < 50; row++ { // the close of the next segment is not drawn back into this one
		if g[4][row] == g[4][row] {
			t.Fatalf("chikou on row %d took a close from the next segment", row)
		}
	}
	if g[4][50-d-1] != cs[50-1] {
		t.Fatalf("chikou on row %d is %v, want the last close of its own segment %v", 50-d-1, g[4][50-d-1], cs[49])
	}
	if g[0][50] == g[0][50] || g[0][50+3] == g[0][50+3] || g[0][50+4] != g[0][50+4] {
		t.Fatalf("tenkan did not restart on row 50: %v", g[0][48:58])
	}
}

func TestIchimokuNoSegmentStartsChangesNothing(t *testing.T) {
	hs, ls, cs := ichimokuBars(90)
	plain, err := Ichimoku(prices(hs...), prices(ls...), prices(cs...), 9, 26, 52)
	if err != nil {
		t.Fatal(err)
	}
	for _, starts := range [][]int{nil, {}, {0}} {
		same, err := Ichimoku(prices(hs...), prices(ls...), prices(cs...), 9, 26, 52, IchimokuOptions{SegmentStarts: starts})
		if err != nil {
			t.Fatal(err)
		}
		a, b := ichimokuLines(t, plain), ichimokuLines(t, same)
		for li := range a {
			if !sameFloats(a[li], b[li]) {
				t.Fatalf("starts %v changed line %d", starts, li)
			}
		}
	}
}

func TestIchimokuSegmentStartsAndCloseAreChecked(t *testing.T) {
	hs, ls, cs := ichimokuBars(30)
	for _, bad := range [][]int{{30}, {-1}, {99}} {
		if _, err := Ichimoku(prices(hs...), prices(ls...), prices(cs...), 5, 10, 20, IchimokuOptions{SegmentStarts: bad}); err == nil {
			t.Errorf("starts %v accepted", bad)
		}
	}
	badClose := append([]float64(nil), cs...)
	badClose[7] = math.NaN()
	if _, err := Ichimoku(prices(hs...), prices(ls...), prices(badClose...), 5, 10, 20); err == nil {
		t.Error("a NaN close was accepted")
	}
	if _, err := Ichimoku(prices(hs...), prices(ls...), prices(cs[:20]...), 5, 10, 20); err == nil {
		t.Error("a shorter close was accepted")
	}
	// A segment shorter than the displacement shows only NaN in the cloud and in chikou.
	r, err := Ichimoku(prices(hs...), prices(ls...), prices(cs...), 2, 3, 4, IchimokuOptions{Displacement: 6, SegmentStarts: []int{27}})
	if err != nil {
		t.Fatal(err)
	}
	g := ichimokuLines(t, r)
	for _, row := range []int{27, 28, 29} {
		if g[2][row] == g[2][row] || g[4][row] == g[4][row] {
			t.Errorf("row %d of a 3-row segment shows a value", row)
		}
	}
}

func TestTRIXSignalSurvivesAnUndefinedRow(t *testing.T) {
	nan := math.NaN()
	// Row 1 has no TRIX because the triple EMA before it is 0; the signal is seeded from the first two defined values.
	r, err := TRIX(prices(0, 1, 2, 3, 4), 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	want := []float64{nan, nan, nan, 75, 75 + (100.0/3-75)*2/3}
	if got := floats(t, r.Signal); !sameFloats(got, want) {
		t.Fatalf("signal %v, want %v", got, want)
	}
	// A stretch of zero prices leaves TRIX undefined; the signal restarts from the next full window instead of staying NaN for good.
	r, err = TRIX(prices(1, 2, 3, 0, 0, 0, 5, 6, 7, 8), 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	got := floats(t, r.Signal)
	if got[2] != 75 || got[4] == got[4] || got[7] == got[7] {
		t.Fatalf("signal around the gap: %v", got)
	}
	if want := (20 + 100.0/6) / 2; math.Abs(got[8]-want) > 1e-12 {
		t.Fatalf("signal after the gap %v, want %v", got[8], want)
	}
}
