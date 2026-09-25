package nn

import (
	"math"
	"math/rand"
	"testing"
)

// TestTanhHighPrecisionAgreesWithOracle checks the fallback over the range
// where the analytic cutoffs leave the result to the high-precision path.
func TestTanhHighPrecisionAgreesWithOracle(t *testing.T) {
	rng := rand.New(rand.NewSource(29))
	lower := float32(math.Ldexp(1, -13))
	upper := float32(9.5)

	for index := 0; index < 5000; index++ {
		x := lower + float32(rng.Float64())*(upper-lower)
		if rng.Intn(2) != 0 {
			x = -x
		}
		got := math.Float32bits(tanhHighPrecision(x))
		want := math.Float32bits(tanhOracle(x))
		if got != want {
			t.Fatalf("sample %d: input bits %#08x, result bits %#08x, want %#08x", index, math.Float32bits(x), got, want)
		}
	}
}

// TestTanhFloat32AgreesWithOracleOnSamples checks all special and analytic
// branches as well as the high-precision branch against the independent oracle.
func TestTanhFloat32AgreesWithOracleOnSamples(t *testing.T) {
	rng := rand.New(rand.NewSource(31))
	lower := float32(math.Ldexp(1, -13))
	upper := float32(9.5)

	for index := 0; index < 50000; index++ {
		x := math.Float32frombits(rng.Uint32())
		actual := tanhFloat32(x)

		if x != x {
			if actual == actual {
				t.Fatalf("sample %d: NaN input %#08x returned %#08x", index, math.Float32bits(x), math.Float32bits(actual))
			}
			continue
		}

		magnitude := x
		if magnitude < 0 {
			magnitude = -magnitude
		}

		var expected float32
		switch {
		case math.IsInf(float64(x), 0):
			expected = float32(math.Copysign(1, float64(x)))
		case magnitude < lower:
			expected = x
		case magnitude >= upper:
			expected = float32(math.Copysign(1, float64(x)))
		default:
			expected = tanhOracle(x)
		}

		got := math.Float32bits(actual)
		want := math.Float32bits(expected)
		if got != want {
			t.Fatalf("sample %d: input bits %#08x, result bits %#08x, want %#08x", index, math.Float32bits(x), got, want)
		}
	}
}

// TestTanhFloat32FallsBackNearMidpoints proves that the conservative fast-path
// test actually exercises the independent fallback and that every fallback
// result still agrees with the oracle.
func TestTanhFloat32FallsBackNearMidpoints(t *testing.T) {
	const (
		startBits uint32 = 0x3f000000
		count            = 1 << 20
	)
	fallbackCount := 0
	for offset := uint32(0); offset < count; offset++ {
		x := math.Float32frombits(startBits + offset)
		if !tanhNeedsHighPrecisionForTest(x) {
			continue
		}
		fallbackCount++
		got := math.Float32bits(tanhFloat32(x))
		want := math.Float32bits(tanhOracle(x))
		if got != want {
			t.Fatalf("input bits %#08x: result bits %#08x, want %#08x", startBits+offset, got, want)
		}
	}
	if fallbackCount == 0 {
		t.Fatal("the midpoint scan found no high-precision fallbacks")
	}
	t.Logf("tanh high-precision fallback count: %d", fallbackCount)
}

func TestTanhVJPRoundsEveryStep(t *testing.T) {
	const count = 100000

	r := rand.New(rand.NewSource(37))
	outputData := make([]float32, count)
	upstreamData := make([]float32, count)
	want := make([]float32, count)
	for index := range outputData {
		y := float32(r.Float64()*2 - 1)
		u := float32(math.Ldexp(r.NormFloat64(), r.Intn(41)-20))
		outputData[index] = y
		upstreamData[index] = u
		s := float32(float64(y) * float64(y))
		c := float32(1 - float64(s))
		want[index] = float32(float64(u) * float64(c))
	}

	shape := []int{count}
	output, err := newFloat32Tensor(shape, outputData)
	if err != nil {
		t.Fatal(err)
	}
	upstream, err := newFloat32Tensor(shape, upstreamData)
	if err != nil {
		t.Fatal(err)
	}
	gradient := tanhVJP(output, upstream)

	mismatches := 0
	firstMismatch := -1
	for index := range want {
		if math.Float32bits(gradient.data[index]) != math.Float32bits(want[index]) {
			mismatches++
			if firstMismatch == -1 {
				firstMismatch = index
			}
		}
	}
	if mismatches != 0 {
		t.Fatalf("tanhVJP mismatches: %d; first at index %d: result bits %#08x, want %#08x", mismatches, firstMismatch, math.Float32bits(gradient.data[firstMismatch]), math.Float32bits(want[firstMismatch]))
	}
}

// TestTanhFloat64SettlesAtMidpoints pins the integer midpoint test at the
// float64 steps around one float32 midpoint, where the decision flips.
func TestTanhFloat64SettlesAtMidpoints(t *testing.T) {
	f := float32(0.75)
	mid := (float64(f) + float64(math.Nextafter32(f, 1))) / 2
	offsets := []int{0, 1, 1 << 13, (1 << 13) + 1, -(1 << 13), -((1 << 13) + 1)}

	for _, k := range offsets {
		bits := math.Float64bits(mid)
		if k < 0 {
			bits -= uint64(-k)
		} else {
			bits += uint64(k)
		}
		want := k > 1<<13 || k < -(1<<13)
		if got := tanhFloat64Settles(math.Float64frombits(bits)); got != want {
			t.Errorf("offset %d: bits %#016x, tanhFloat64Settles = %t, want %t", k, bits, got, want)
		}
	}

	if got := tanhFloat64Settles(float64(f)); !got {
		t.Errorf("tanhFloat64Settles(float64(0.75)) = %t, want true", got)
	}
	if got := tanhFloat64Settles(-mid); got {
		t.Errorf("tanhFloat64Settles(-mid) = %t, want false", got)
	}
}

// tanhNeedsHighPrecisionForTest reports whether tanhFloat32 sends x to its
// high-precision path, by the same criterion tanhFloat32 applies.
func tanhNeedsHighPrecisionForTest(x float32) bool {
	if x != x || math.IsInf(float64(x), 0) {
		return false
	}
	a := x
	if a < 0 {
		a = -a
	}
	if a < float32(math.Ldexp(1, -13)) || a >= 9.5 {
		return false
	}

	t := math.Tanh(float64(x))
	return !tanhFloat64Settles(t)
}
