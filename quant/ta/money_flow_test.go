package ta

import (
	"encoding/json"
	"math"
	"os"
	"testing"
)

type volumeCase struct {
	indicatorCase
	Volume []float64
}

func volumeCases(t *testing.T) []volumeCase {
	t.Helper()
	raw, err := os.ReadFile("testdata/volume_baseline.json")
	if err != nil {
		t.Fatal(err)
	}
	var f struct {
		Cases []struct {
			High, Low, Close, Volume []float64
			Expected                 map[string]json.RawMessage
		}
	}
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	var out []volumeCase
	for _, c := range f.Cases {
		out = append(out, volumeCase{indicatorCase{High: c.High, Low: c.Low, Close: c.Close, Expected: c.Expected}, c.Volume})
	}
	return out
}

func TestMoneyFlowMatchesTheReference(t *testing.T) {
	for ci, c := range volumeCases(t) {
		hi, lo, cl, vo := prices(c.High...), prices(c.Low...), prices(c.Close...), prices(c.Volume...)
		name := func(s string) string { return s + " case " + itoa(ci) }
		for _, n := range []int{14, 5} {
			got, err := MFI(hi, lo, cl, vo, n)
			if err != nil {
				t.Fatal(err)
			}
			sameAsReference(t, name("mfi"), got, c.series(t, "mfi"+itoa(n)))
		}
		ad, err := AccumulationDistribution(hi, lo, cl, vo)
		if err != nil {
			t.Fatal(err)
		}
		sameAsReference(t, name("ad"), ad, c.series(t, "ad"))
		for _, s := range [][2]int{{3, 10}, {2, 5}} {
			got, err := ChaikinOscillator(hi, lo, cl, vo, s[0], s[1])
			if err != nil {
				t.Fatal(err)
			}
			sameAsReference(t, name("adosc"), got, c.series(t, "adosc"+itoa(s[0])+"_"+itoa(s[1])))
		}
		for _, n := range []int{20, 5} {
			got, err := CMF(hi, lo, cl, vo, n)
			if err != nil {
				t.Fatal(err)
			}
			sameAsReference(t, name("cmf"), got, c.series(t, "cmf"+itoa(n)))
		}
		for _, n := range []int{14, 3} {
			got, err := EaseOfMovement(hi, lo, vo, n)
			if err != nil {
				t.Fatal(err)
			}
			sameAsReference(t, name("emv"), got, c.series(t, "emv"+itoa(n)))
		}
		scaled, err := EaseOfMovement(hi, lo, vo, 14, EaseOfMovementOptions{Scale: 1e6})
		if err != nil {
			t.Fatal(err)
		}
		sameAsReference(t, name("emv scale"), scaled, c.series(t, "emv14_scale1e6"))
		nvi, _ := NVI(cl, vo)
		pvi, _ := PVI(cl, vo)
		sameAsReference(t, name("nvi"), nvi, c.series(t, "nvi"))
		sameAsReference(t, name("pvi"), pvi, c.series(t, "pvi"))
		for _, s := range []struct {
			n, bins int
			va      float64
			key     string
		}{{20, 24, 0.7, "profile20_24_0.7"}, {10, 12, 0.5, "profile10_12_0.5"}} {
			got, err := VolumeProfile(hi, lo, cl, vo, s.n, VolumeProfileOptions{Bins: s.bins, ValueArea: s.va})
			if err != nil {
				t.Fatal(err)
			}
			want := c.lines(t, s.key)
			sameAsReference(t, name(s.key+" poc"), got.POC, want["poc"])
			sameAsReference(t, name(s.key+" vah"), got.ValueAreaHigh, want["vah"])
			sameAsReference(t, name(s.key+" val"), got.ValueAreaLow, want["val"])
		}
	}
}

func TestMoneyFlowByHand(t *testing.T) {
	// A close at the high adds the whole volume, at the low subtracts it, and a bar with no range adds nothing.
	ad, err := AccumulationDistribution(prices(10, 10, 10), prices(8, 8, 10), prices(10, 8, 10), prices(100, 40, 70))
	if err != nil {
		t.Fatal(err)
	}
	if g := floats(t, ad); g[0] != 100 || g[1] != 60 || g[2] != 60 {
		t.Fatalf("a/d %v", g)
	}
	nvi, _ := NVI(prices(10, 11, 12, 6), prices(100, 50, 60, 20))
	pvi, _ := PVI(prices(10, 11, 12, 6), prices(100, 50, 60, 20))
	// Volume falls on row 1 (NVI moves 10%), rises on row 2 (PVI moves), falls on row 3.
	g, p := floats(t, nvi), floats(t, pvi)
	if g[0] != 1000 || math.Abs(g[1]-1100) > 1e-9 || math.Abs(g[2]-1100) > 1e-9 || math.Abs(g[3]-550) > 1e-9 {
		t.Fatalf("nvi %v", g)
	}
	if p[0] != 1000 || p[1] != 1000 || math.Abs(p[2]-1000*12.0/11) > 1e-9 || math.Abs(p[3]-p[2]) > 1e-9 {
		t.Fatalf("pvi %v", p)
	}
	// With no price range or no volume the profile is undefined.
	flat := prices(5, 5, 5)
	vp, err := VolumeProfile(flat, flat, flat, prices(1, 1, 1), 2)
	if err != nil || !math.IsNaN(floats(t, vp.POC)[2]) {
		t.Fatalf("a flat window has no profile: %v", err)
	}
	// Volume at one price level only: that level is the point of control and the value area.
	h2, l2, c2 := prices(10, 10, 12, 10), prices(10, 10, 10, 10), prices(10, 10, 10, 10)
	vp, err = VolumeProfile(h2, l2, c2, prices(0, 0, 9, 0), 4, VolumeProfileOptions{Bins: 2, ValueArea: 0.5})
	if err != nil {
		t.Fatal(err)
	}
	// Bar 2 spans 10 to 12, so its volume splits evenly into the two bins; the lower bin wins the tie.
	if g := floats(t, vp.POC)[3]; g != 10.5 {
		t.Fatalf("POC %v, want the middle of the lower bin 10.5", g)
	}
}

func TestMoneyFlowRefuseBadInput(t *testing.T) {
	p := prices(1, 2, 3, 4, 5, 6, 7, 8)
	neg := prices(1, -1, 1, 1, 1, 1, 1, 1)
	for name, err := range map[string]error{
		"mfi n":      second(MFI(p, p, p, p, 0)),
		"mfi volume": second(MFI(p, p, p, neg, 2)),
		"mfi len":    second(MFI(p, p, p, prices(1, 2), 2)),
		"ad len":     second(AccumulationDistribution(p, p, p, prices(1))),
		"cmf n":      second(CMF(p, p, p, p, 0)),
		"chaikin":    second(ChaikinOscillator(p, p, p, p, 5, 5)),
		"chaikin f":  second(ChaikinOscillator(p, p, p, p, 0, 5)),
		"emv n":      second(EaseOfMovement(p, p, p, 0)),
		"emv scale":  second(EaseOfMovement(p, p, p, 2, EaseOfMovementOptions{Scale: -1})),
		"emv two":    second(EaseOfMovement(p, p, p, 2, EaseOfMovementOptions{}, EaseOfMovementOptions{})),
		"nvi close":  second(NVI(prices(1, 0, 1), prices(1, 1, 1))),
		"nvi len":    second(NVI(p, prices(1))),
		"vp n":       vpErr(VolumeProfile(p, p, p, p, 0)),
		"vp bins":    vpErr(VolumeProfile(p, p, p, p, 2, VolumeProfileOptions{Bins: -1})),
		"vp va":      vpErr(VolumeProfile(p, p, p, p, 2, VolumeProfileOptions{ValueArea: 1.5})),
		"vp two":     vpErr(VolumeProfile(p, p, p, p, 2, VolumeProfileOptions{}, VolumeProfileOptions{})),
	} {
		if err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
	if _, err := CMF(nil, p, p, p, 2); err == nil {
		t.Error("nil high")
	}
}

func vpErr(_ *VolumeProfileResult, err error) error { return err }

func TestMoneyFlowListsAreNamed(t *testing.T) {
	p := prices(1, 3, 2, 5, 4, 6, 5, 8, 7, 9, 8, 11, 10, 12, 11, 14)
	v := prices(10, 20, 15, 30, 25, 35, 20, 40, 30, 50, 35, 45, 40, 60, 50, 70)
	mfi, _ := MFI(p, p, p, v, 3)
	ad, _ := AccumulationDistribution(p, p, p, v)
	cmf, _ := CMF(p, p, p, v, 3)
	ch, _ := ChaikinOscillator(p, p, p, v, 2, 3)
	em, _ := EaseOfMovement(p, p, v, 3)
	nv, _ := NVI(p, v)
	pv, _ := PVI(p, v)
	vp, _ := VolumeProfile(p, p, p, v, 3)
	for want, got := range map[string]string{
		"MFI": mfi.GetName(), "AccumulationDistribution": ad.GetName(), "CMF": cmf.GetName(),
		"ChaikinOscillator": ch.GetName(), "EaseOfMovement": em.GetName(), "NVI": nv.GetName(), "PVI": pv.GetName(),
		"POC": vp.POC.GetName(), "ValueAreaHigh": vp.ValueAreaHigh.GetName(), "ValueAreaLow": vp.ValueAreaLow.GetName(),
	} {
		if got != want {
			t.Errorf("a list is named %q, want %q", got, want)
		}
	}
}
