package algorithms

import (
	"math"
	"testing"
)

// IN-3: routing every integer through float64 makes any two int64 above 2^53
// compare equal, so Sort, Rank, SortBy and Describe's min/max all go wrong.
func TestCompareAnyExactForLargeIntegers(t *testing.T) {
	const twoP53 = int64(1) << 53
	cases := []struct {
		name string
		a, b any
		want int
	}{
		{"int64 just above 2^53", twoP53 + 1, twoP53, 1},
		{"int64 two apart above 2^53", twoP53, twoP53 + 2, -1},
		{"int64 equal", twoP53 + 1, twoP53 + 1, 0},
		{"max int64 vs one less", int64(math.MaxInt64), int64(math.MaxInt64 - 1), 1},
		{"uint64 beyond int64", uint64(math.MaxUint64), int64(math.MaxInt64), 1},
		{"int64 vs uint64 beyond int64", int64(math.MaxInt64), uint64(math.MaxUint64), -1},
		{"negative int64 vs uint64", int64(-1), uint64(0), -1},
		{"mixed widths", int32(5), int64(5), 0},
		{"int vs float still works", 2, 2.5, -1},
		{"float vs int still works", 3.5, 3, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := CompareAny(c.a, c.b); got != c.want {
				t.Fatalf("CompareAny(%v, %v) = %d, want %d", c.a, c.b, got, c.want)
			}
			if got := CompareAny(c.b, c.a); got != -c.want {
				t.Fatalf("CompareAny(%v, %v) = %d, want %d (antisymmetry)", c.b, c.a, got, -c.want)
			}
		})
	}
}

// IN-4: the implementation used the first-degree Lagrange basis, so it was not
// a Hermite interpolant at all — the derivative conditions were not met.
func TestHermiteInterpolationSatisfiesItsConditions(t *testing.T) {
	// Nodes are 0..n-1. p(0)=0, p'(0)=1, p(1)=0, p'(1)=0 has the unique cubic
	// solution p(t) = t(1-t)^2, so p(0.5) = 0.125.
	got, err := HermiteInterpolation([]float64{0, 0}, []float64{1, 0}, 0.5)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(got-0.125) > 1e-12 {
		t.Fatalf("p(0.5) = %v, want 0.125", got)
	}

	// It must reproduce a polynomial of low enough degree exactly: x^2 on
	// nodes 0,1,2 with the true derivatives 2x.
	data := []float64{0, 1, 4}
	derivs := []float64{0, 2, 4}
	for _, x := range []float64{0, 0.25, 0.5, 1, 1.5, 2} {
		got, err := HermiteInterpolation(data, derivs, x)
		if err != nil {
			t.Fatal(err)
		}
		if math.Abs(got-x*x) > 1e-9 {
			t.Fatalf("p(%v) = %v, want %v", x, got, x*x)
		}
	}

	// The interpolant passes through every node value and matches every node
	// derivative (checked numerically).
	const h = 1e-6
	for i := range data {
		xi := float64(i)
		at, err := HermiteInterpolation(data, derivs, xi)
		if err != nil {
			t.Fatal(err)
		}
		if math.Abs(at-data[i]) > 1e-9 {
			t.Fatalf("p(%v) = %v, want the node value %v", xi, at, data[i])
		}
		lo, _ := HermiteInterpolation(data, derivs, xi-h)
		hi, _ := HermiteInterpolation(data, derivs, xi+h)
		if d := (hi - lo) / (2 * h); math.Abs(d-derivs[i]) > 1e-4 {
			t.Fatalf("p'(%v) = %v, want %v", xi, d, derivs[i])
		}
	}
}
