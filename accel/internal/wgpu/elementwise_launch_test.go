package wgpu

import (
	"context"
	"math"
	"testing"
)

// TestElementwiseLaunchesCoverEveryElement runs kernels over arrays longer
// than their launch span, so each takes several launches, and checks that
// every element was written with the value of its own index.
func TestElementwiseLaunchesCoverEveryElement(t *testing.T) {
	requireResidentDevice(t)
	const period = 997
	pattern := func(i int) float32 { return float32(i%period)/period*4 - 2 + 0.001 }
	upload := func(n int, f func(int) float32) *Buffer {
		values := make([]float32, n)
		for i := range values {
			values[i] = f(i)
		}
		b, err := Upload(values)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	download := func(b *Buffer) []float32 {
		got, err := b.Download(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		return got
	}

	// A fixed-point kernel: tanh over two and a bit spans.
	n := 2*fixedLaunchSpan + 7
	a := upload(n, pattern)
	defer a.Release()
	out, err := TanhRounded(a)
	if err != nil {
		t.Fatal(err)
	}
	got := download(out)
	out.Release()
	for i := period; i < n; i++ {
		if math.Float32bits(got[i]) != math.Float32bits(got[i%period]) || got[i%period] == 0 && pattern(i) != 0 {
			t.Fatalf("tanh[%d] = %#x, want tanh[%d] = %#x", i, math.Float32bits(got[i]), i%period, math.Float32bits(got[i%period]))
		}
	}

	// An exact-register kernel with a broadcast operand: the add reads
	// b[i % len(b)] at the element's own index in every launch.
	n = 7 * (2*exactLaunchSpan/7 + 3) // a multiple of the broadcast length, past two spans
	x := upload(n, func(i int) float32 { return float32(i) })
	defer x.Release()
	y := upload(7, func(i int) float32 { return 0.5 + float32(i) })
	defer y.Release()
	sum, err := AddRounded(x, y)
	if err != nil {
		t.Fatal(err)
	}
	got = download(sum)
	sum.Release()
	for i := range got {
		if want := float32(i) + (0.5 + float32(i%7)); got[i] != want {
			t.Fatalf("add[%d] = %g, want %g", i, got[i], want)
		}
	}

	// An in-place kernel: SGD changes every element exactly once.
	w := upload(n, func(int) float32 { return 1 })
	defer w.Release()
	g := upload(n, func(int) float32 { return 2 })
	defer g.Release()
	if err := SGDRounded(w, g, math.Float32bits(-0.25)); err != nil {
		t.Fatal(err)
	}
	for i, v := range download(w) {
		if v != 0.5 {
			t.Fatalf("w[%d] = %g after one step, want 0.5", i, v)
		}
	}
}
