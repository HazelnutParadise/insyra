package wgpu

import (
	"context"
	"math"
	"math/rand"
	"testing"
)

// tanhGradReference is RN(u * RN(1 - RN(y*y))) in Go float32, each zero +0.
func tanhGradReference(y, u float32) float32 {
	pz := func(x float32) float32 {
		if x == 0 {
			return 0
		}
		return x
	}
	square := pz(float32(y * y))
	complement := pz(float32(1 - square))
	return pz(float32(u * complement))
}

func TestTanhGradRoundedMatchesFloat32Steps(t *testing.T) {
	requireResidentDevice(t)
	bits := math.Float32frombits
	negZero := float32(math.Copysign(0, -1))
	special := []float32{0, negZero, 1, -1, bits(0x3f7fffff), bits(0xbf7fffff), bits(1), bits(0x80000001),
		bits(0x1f800000), 0.5, -0.5, 2, 1e20, float32(math.Inf(1)), float32(math.Inf(-1)), float32(math.NaN())}
	var ys, us []float32
	for _, y := range special {
		for _, u := range special {
			ys, us = append(ys, y), append(us, u)
		}
	}
	r := rand.New(rand.NewSource(38))
	for i := 0; i < 20000; i++ {
		ys = append(ys, float32(math.Tanh(r.NormFloat64()*3)))
		us = append(us, float32(r.NormFloat64()*math.Ldexp(1, r.Intn(300)-150)))
	}
	yb, err := Upload(ys)
	if err != nil {
		t.Fatal(err)
	}
	defer yb.Release()
	ub, err := Upload(us)
	if err != nil {
		t.Fatal(err)
	}
	defer ub.Release()
	out, err := TanhGradRounded(yb, ub)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Release()
	got, err := out.Download(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	bad := 0
	for i := range ys {
		want := tanhGradReference(ys[i], us[i])
		if math.IsNaN(float64(want)) && math.IsNaN(float64(got[i])) {
			continue
		}
		if math.Float32bits(got[i]) != math.Float32bits(want) {
			if bad < 12 {
				t.Errorf("y=%g u=%g: %#x, want %#x", ys[i], us[i], math.Float32bits(got[i]), math.Float32bits(want))
			}
			bad++
		}
	}
	if bad > 0 {
		t.Errorf("%d of %d wrong", bad, len(ys))
	}
	short, err := Upload([]float32{1})
	if err != nil {
		t.Fatal(err)
	}
	defer short.Release()
	if _, err := TanhGradRounded(yb, short); err == nil {
		t.Error("a length mismatch was accepted")
	}
}
