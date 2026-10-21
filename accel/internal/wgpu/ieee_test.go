package wgpu

import (
	"context"
	"math"
	"math/rand"
	"os"
	"testing"
)

func f32(bits uint32) float32 { return math.Float32frombits(bits) }

// interesting float32 bit patterns around every boundary division and square
// root care about.
func ieeeEdgePatterns() []uint32 {
	var out []uint32
	for _, exp := range []uint32{0, 1, 2, 3, 100, 126, 127, 128, 150, 253, 254, 255} {
		for _, frac := range []uint32{0, 1, 2, 3, 0x400000, 0x7ffffe, 0x7fffff, 0x555555, 0x2aaaaa} {
			out = append(out, exp<<23|frac, 0x80000000|exp<<23|frac)
		}
	}
	return out
}

func sameResult(got, want float32) bool {
	if math.IsNaN(float64(want)) {
		return math.IsNaN(float64(got))
	}
	return math.Float32bits(got) == math.Float32bits(want)
}

func TestDivAndSqrtRoundedMatchTheHost(t *testing.T) {
	requireResidentDevice(t)
	r := rand.New(rand.NewSource(21))
	edges := ieeeEdgePatterns()
	var a, b []float32
	for _, x := range edges {
		for _, y := range edges {
			a = append(a, f32(x))
			b = append(b, f32(y))
		}
	}
	for i := 0; i < 2_000_000; i++ {
		a = append(a, f32(r.Uint32()))
		b = append(b, f32(r.Uint32()))
	}
	// near-equal operands exercise the quotient close to 1 and near ties
	for i := 0; i < 500_000; i++ {
		x := f32(r.Uint32())
		a = append(a, x)
		b = append(b, f32(math.Float32bits(x)+uint32(r.Intn(5))-2))
	}
	ab, err := Upload(a)
	if err != nil {
		t.Fatal(err)
	}
	defer ab.Release()
	bb, _ := Upload(b)
	defer bb.Release()
	q, err := DivRounded(ab, bb)
	if err != nil {
		t.Fatal(err)
	}
	defer q.Release()
	got, err := q.Download(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for i := range a {
		if want := a[i] / b[i]; !sameResult(got[i], want) {
			t.Fatalf("%g / %g (%#x / %#x) = %g (%#x), want %g (%#x)", a[i], b[i], math.Float32bits(a[i]), math.Float32bits(b[i]),
				got[i], math.Float32bits(got[i]), want, math.Float32bits(want))
		}
	}

	s, err := SqrtRounded(ab)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Release()
	gotSqrt, err := s.Download(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for i := range a {
		if want := float32(math.Sqrt(float64(a[i]))); !sameResult(gotSqrt[i], want) {
			t.Fatalf("sqrt(%g) (%#x) = %g (%#x), want %g (%#x)", a[i], math.Float32bits(a[i]), gotSqrt[i], math.Float32bits(gotSqrt[i]), want, math.Float32bits(want))
		}
	}
}

// TestSqrtRoundedIsExhaustive checks every float32 input against the host,
// in chunks. It runs only when INSYRA_ACCEL_EXHAUSTIVE=1.
func TestSqrtRoundedIsExhaustive(t *testing.T) {
	requireResidentDevice(t)
	if os.Getenv("INSYRA_ACCEL_EXHAUSTIVE") != "1" {
		t.Skip("set INSYRA_ACCEL_EXHAUSTIVE=1")
	}
	const chunk = 1 << 24
	in := make([]float32, chunk)
	for base := uint64(0); base < 1<<32; base += chunk {
		for i := range in {
			in[i] = f32(uint32(base) + uint32(i))
		}
		ab, err := Upload(in)
		if err != nil {
			t.Fatal(err)
		}
		s, err := SqrtRounded(ab)
		if err != nil {
			t.Fatal(err)
		}
		got, err := s.Download(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		for i := range in {
			if want := float32(math.Sqrt(float64(in[i]))); !sameResult(got[i], want) {
				t.Fatalf("sqrt(%#x) = %#x, want %#x", math.Float32bits(in[i]), math.Float32bits(got[i]), math.Float32bits(want))
			}
		}
		ab.Release()
		s.Release()
	}
}
