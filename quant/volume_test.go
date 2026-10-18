package quant

import (
	"math"
	"reflect"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

func floats(t *testing.T, dl *insyra.DataList) []float64 {
	t.Helper()
	out := make([]float64, 0, dl.Len())
	for _, v := range dl.Data() {
		f, ok := insyra.ToFloat64Safe(v)
		if !ok {
			t.Fatalf("not a number: %v", v)
		}
		out = append(out, f)
	}
	return out
}

func TestOBV(t *testing.T) {
	got, err := OBV(insyra.NewDataList(10.0, 11.0, 11.0, 10.0, 12.0), insyra.NewDataList(100.0, 200.0, 300.0, 400.0, 500.0))
	if err != nil {
		t.Fatal(err)
	}
	if want := []float64{0, 200, 200, -200, 300}; !reflect.DeepEqual(floats(t, got), want) {
		t.Errorf("got %v, want %v", floats(t, got), want)
	}
}

func TestOBVFromReturnsUsesTheGivenDirection(t *testing.T) {
	// An ex-dividend day: the raw close falls but the adjusted return is positive.
	got, err := OBVFromReturns(insyra.NewDataList(0.01, 0.02, -0.01, 0.0, 0.03), insyra.NewDataList(1.0, 2.0, 3.0, 4.0, 5.0))
	if err != nil {
		t.Fatal(err)
	}
	if want := []float64{0, 2, -1, -1, 4}; !reflect.DeepEqual(floats(t, got), want) {
		t.Errorf("got %v, want %v", floats(t, got), want)
	}
}

func TestSignedVolumeRatio(t *testing.T) {
	got, err := SignedVolumeRatio(insyra.NewDataList(1.0, -1.0, 1.0, 0.0), insyra.NewDataList(1.0, 2.0, 3.0, 4.0), 2)
	if err != nil {
		t.Fatal(err)
	}
	g := floats(t, got)
	if !math.IsNaN(g[0]) || math.Abs(g[1]+1.0/3) > 1e-15 || math.Abs(g[2]-0.2) > 1e-15 || math.Abs(g[3]-3.0/7) > 1e-15 {
		t.Errorf("got %v", g)
	}
	zero, err := SignedVolumeRatio(insyra.NewDataList(1.0, 1.0), insyra.NewDataList(0.0, 0.0), 2)
	if err != nil || !math.IsNaN(floats(t, zero)[1]) {
		t.Errorf("a window with no volume should be NaN: %v, %v", zero, err)
	}
}

func TestDivergence(t *testing.T) {
	bear, _ := Divergence(insyra.NewDataList(1.0, 2.0, 3.0, 4.0), insyra.NewDataList(1.0, 2.0, 3.0, 2.0), 2, 0)
	if want := []float64{0, 0, 0, -1}; !reflect.DeepEqual(floats(t, bear), want) {
		t.Errorf("bearish: got %v, want %v", floats(t, bear), want)
	}
	bull, _ := Divergence(insyra.NewDataList(5.0, 4.0, 3.0, 2.0), insyra.NewDataList(5.0, 4.0, 3.0, 4.0), 2, 0)
	if want := []float64{0, 0, 0, 1}; !reflect.DeepEqual(floats(t, bull), want) {
		t.Errorf("bullish: got %v, want %v", floats(t, bull), want)
	}
	p, ind := insyra.NewDataList(1.0, 2.0, 3.0, 4.0, 5.0), insyra.NewDataList(1.0, 2.0, 3.0, 2.0, 1.0)
	every, _ := Divergence(p, ind, 2, 0)
	if want := []float64{0, 0, 0, -1, -1}; !reflect.DeepEqual(floats(t, every), want) {
		t.Errorf("no cooldown: got %v, want %v", floats(t, every), want)
	}
	once, _ := Divergence(p, ind, 2, 1)
	if want := []float64{0, 0, 0, -1, 0}; !reflect.DeepEqual(floats(t, once), want) {
		t.Errorf("cooldown 1: got %v, want %v", floats(t, once), want)
	}
}

func TestVolumeFunctionsRefuseBadInput(t *testing.T) {
	good := insyra.NewDataList(1.0, 2.0, 3.0)
	var nilList *insyra.DataList
	cases := map[string]func() error{
		"NaN close":         func() error { _, e := OBV(insyra.NewDataList(1.0, math.NaN(), 3.0), good); return e },
		"Inf volume":        func() error { _, e := OBV(good, insyra.NewDataList(1.0, math.Inf(1), 3.0)); return e },
		"negative volume":   func() error { _, e := OBV(good, insyra.NewDataList(1.0, -2.0, 3.0)); return e },
		"length mismatch":   func() error { _, e := OBV(good, insyra.NewDataList(1.0, 2.0)); return e },
		"nil close":         func() error { _, e := OBV(nilList, good); return e },
		"returns mismatch":  func() error { _, e := OBVFromReturns(good, insyra.NewDataList(1.0)); return e },
		"ratio window zero": func() error { _, e := SignedVolumeRatio(good, good, 0); return e },
		"divergence n":      func() error { _, e := Divergence(good, good, 0, 0); return e },
		"divergence cool":   func() error { _, e := Divergence(good, good, 1, -1); return e },
	}
	for name, f := range cases {
		if f() == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}
