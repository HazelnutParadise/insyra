package ta

import (
	"encoding/json"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

type oscillatorCase struct {
	indicatorCase
	Open, Volume []float64
}

func oscillatorCases(t *testing.T) []oscillatorCase {
	t.Helper()
	raw, err := os.ReadFile("testdata/oscillators_baseline.json")
	if err != nil {
		t.Fatal(err)
	}
	var f struct {
		Cases []struct {
			Open, High, Low, Close, Volume []float64
			Expected                       map[string]json.RawMessage
		}
	}
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	var out []oscillatorCase
	for _, c := range f.Cases {
		out = append(out, oscillatorCase{indicatorCase{High: c.High, Low: c.Low, Close: c.Close, Expected: c.Expected}, c.Open, c.Volume})
	}
	return out
}

func TestOscillatorsMatchTheReference(t *testing.T) {
	for ci, c := range oscillatorCases(t) {
		op, hi, lo, cl, vo := prices(c.Open...), prices(c.High...), prices(c.Low...), prices(c.Close...), prices(c.Volume...)
		name := func(s string) string { return s + " case " + itoa(ci) }
		for _, n := range []int{20, 5} {
			got, err := CCI(hi, lo, cl, n)
			if err != nil {
				t.Fatal(err)
			}
			sameAsReference(t, name("cci"), got, c.series(t, "cci"+itoa(n)))
			m, _ := Momentum(cl, n)
			sameAsReference(t, name("momentum"), m, c.series(t, "mom"+itoa(n)))
			r, _ := ROC(cl, n)
			sameAsReference(t, name("roc"), r, c.series(t, "roc"+itoa(n)))
		}
		for _, s := range [][3]int{{7, 14, 28}, {3, 5, 8}} {
			got, err := UltimateOscillator(hi, lo, cl, s[0], s[1], s[2])
			if err != nil {
				t.Fatal(err)
			}
			sameAsReference(t, name("ultosc"), got, c.series(t, "ultosc"+itoa(s[0])+"_"+itoa(s[1])+"_"+itoa(s[2])))
		}
		for _, s := range [][2]int{{14, 14}, {6, 5}} {
			got, err := StochRSI(cl, s[0], s[1], KDOptions{Method: KDMovingAverage, SlowK: 1, SlowD: 3})
			if err != nil {
				t.Fatal(err)
			}
			want := c.lines(t, "stochrsi"+itoa(s[0])+"_"+itoa(s[1]))
			sameWhereDefined(t, name("stochrsi k"), got.K, want["k"], s[0]+s[1]-1)
			sameAsReference(t, name("stochrsi d"), got.D, want["d"])
		}
		for _, n := range []int{26, 6} {
			got, err := ARBR(op, hi, lo, cl, n)
			if err != nil {
				t.Fatal(err)
			}
			want := c.lines(t, "arbr"+itoa(n))
			sameAsReference(t, name("ar"), got.AR, want["ar"])
			sameAsReference(t, name("br"), got.BR, want["br"])
		}
		for kind, m := range map[string]CRMidpoint{"hl": CRMidHighLow, "wc": CRMidWeightedClose, "typ": CRMidTypical, "ohlc": CRMidOHLC} {
			got, err := CR(hi, lo, cl, 26, CROptions{Midpoint: m, Open: op})
			if err != nil {
				t.Fatal(err)
			}
			sameAsReference(t, name("cr "+kind), got, c.series(t, "cr26_"+kind))
		}
		for _, n := range []int{13, 1} {
			got, err := ForceIndex(cl, vo, n)
			if err != nil {
				t.Fatal(err)
			}
			sameAsReference(t, name("force"), got, c.series(t, "force"+itoa(n)))
		}
		for _, n := range []int{13, 5} {
			got, err := ElderRay(hi, lo, cl, n)
			if err != nil {
				t.Fatal(err)
			}
			want := c.lines(t, "elder"+itoa(n))
			sameAsReference(t, name("bull"), got.BullPower, want["bull"])
			sameAsReference(t, name("bear"), got.BearPower, want["bear"])
		}
	}
}

func TestOscillatorsByHand(t *testing.T) {
	m, _ := Momentum(prices(1, 3, 6, 10), 2)
	if g := floats(t, m); g[2] != 5 || g[3] != 7 || !math.IsNaN(g[1]) {
		t.Fatalf("momentum %v", g)
	}
	r, _ := ROC(prices(100, 105, 110), 1)
	if g := floats(t, r); math.Abs(g[1]-5) > 1e-12 || math.Abs(g[2]-(110.0/105-1)*100) > 1e-12 {
		t.Fatalf("roc is a percentage: %v", g)
	}
	// A window of equal typical prices has no deviation.
	flat := prices(5, 5, 5, 5)
	if g := floats(t, mustList(CCI(flat, flat, flat, 3))); !math.IsNaN(g[3]) {
		t.Fatalf("a flat cci is undefined: %v", g)
	}
	// CR's four midpoints differ on the same bars, and OHLC needs the open.
	h, l, c := prices(10, 12, 11, 13, 12, 14), prices(8, 9, 9, 10, 10, 11), prices(9, 11, 10, 12, 11, 13)
	var seen []float64
	for _, mid := range []CRMidpoint{CRMidHighLow, CRMidWeightedClose, CRMidTypical} {
		got, err := CR(h, l, c, 3, CROptions{Midpoint: mid})
		if err != nil {
			t.Fatal(err)
		}
		seen = append(seen, floats(t, got)[5])
	}
	if seen[0] == seen[1] && seen[1] == seen[2] {
		t.Errorf("the midpoints should change CR: %v", seen)
	}
	if _, err := CR(h, l, c, 3, CROptions{Midpoint: CRMidOHLC}); err == nil || !strings.Contains(err.Error(), "Open") {
		t.Errorf("CRMidOHLC without the open: %v", err)
	}
}

func mustList(l *insyra.DataList, err error) *insyra.DataList {
	if err != nil {
		panic(err)
	}
	return l
}

func TestOscillatorsRefuseBadInput(t *testing.T) {
	p := prices(1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12)
	for name, err := range map[string]error{
		"cci n":      second(CCI(p, p, p, 0)),
		"mom n":      second(Momentum(p, 0)),
		"roc n":      second(ROC(p, 0)),
		"ult order":  second(UltimateOscillator(p, p, p, 14, 7, 28)),
		"ult zero":   second(UltimateOscillator(p, p, p, 0, 7, 28)),
		"stochrsi n": secondK(StochRSI(p, 0, 3)),
		"stochrsi s": secondK(StochRSI(p, 3, 0)),
		"stochrsi m": secondK(StochRSI(p, 3, 3, KDOptions{Method: KDMethod(9)})),
		"arbr n":     secondA(ARBR(p, p, p, p, 0)),
		"arbr len":   secondA(ARBR(prices(1), p, p, p, 2)),
		"cr n":       second(CR(p, p, p, 0)),
		"cr mid":     second(CR(p, p, p, 2, CROptions{Midpoint: CRMidpoint(9)})),
		"cr two":     second(CR(p, p, p, 2, CROptions{}, CROptions{})),
		"force n":    second(ForceIndex(p, p, 0)),
		"force len":  second(ForceIndex(p, prices(1, 2), 2)),
		"elder n":    secondE(ElderRay(p, p, p, 0)),
	} {
		if err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
	if _, err := CCI(prices(1, 2), prices(2, 3), prices(1, 2), 2); err == nil {
		t.Error("a high below its low")
	}
	if _, err := CR(nil, p, p, 2); err == nil {
		t.Error("nil high")
	}
}

func second(_ *insyra.DataList, err error) error { return err }
func secondK(_ *KDResult, err error) error       { return err }
func secondA(_ *ARBRResult, err error) error     { return err }
func secondE(_ *ElderRayResult, err error) error { return err }

func TestOscillatorListsAreNamed(t *testing.T) {
	p := prices(1, 3, 2, 5, 4, 6, 5, 8, 7, 9, 8, 11, 10, 12, 11, 14, 13, 15, 14, 17)
	cci, _ := CCI(p, p, p, 3)
	mom, _ := Momentum(p, 2)
	roc, _ := ROC(p, 2)
	ult, _ := UltimateOscillator(p, p, p, 2, 3, 4)
	sr, _ := StochRSI(p, 3, 3)
	ab, _ := ARBR(p, p, p, p, 3)
	cr, _ := CR(p, p, p, 3)
	fi, _ := ForceIndex(p, p, 3)
	er, _ := ElderRay(p, p, p, 3)
	for want, got := range map[string]string{
		"CCI": cci.GetName(), "Momentum": mom.GetName(), "ROC": roc.GetName(), "UltimateOscillator": ult.GetName(),
		"K": sr.K.GetName(), "D": sr.D.GetName(), "AR": ab.AR.GetName(), "BR": ab.BR.GetName(), "CR": cr.GetName(),
		"ForceIndex": fi.GetName(), "BullPower": er.BullPower.GetName(), "BearPower": er.BearPower.GetName(),
	} {
		if got != want {
			t.Errorf("a list is named %q, want %q", got, want)
		}
	}
}
