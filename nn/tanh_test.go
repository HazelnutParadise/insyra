package nn

import (
	"math"
	"math/big"
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
// branches against the independent oracle. The fast path is only sampled here;
// the inputs it cannot decide on its own are the ones in tanhHardCases, and
// TestTanhHardCasesAreHardAndRight and TestTanhIsCorrectlyRoundedExhaustive are
// what cover those.
func TestTanhFloat32AgreesWithOracleOnSamples(t *testing.T) {
	rng := rand.New(rand.NewSource(31))
	lower := float32(math.Ldexp(1, -13))
	upper := float32(9.5)
	inputs := make([]float32, 50000)

	for index := range inputs {
		if index%2 == 0 {
			inputs[index] = math.Float32frombits(rng.Uint32())
			continue
		}
		// Half the samples are drawn log-uniformly over the same range, so the
		// small magnitudes the uniform draw almost never reaches are covered.
		inputs[index] = float32(math.Ldexp(1+rng.Float64(), rng.Intn(17)-13))
	}

	for index, x := range inputs {
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
	offsets := []int{0, 1, tanhSettleSteps, tanhSettleSteps + 1, -tanhSettleSteps, -(tanhSettleSteps + 1)}

	for _, k := range offsets {
		bits := math.Float64bits(mid)
		if k < 0 {
			bits -= uint64(-k)
		} else {
			bits += uint64(k)
		}
		want := k > tanhSettleSteps || k < -tanhSettleSteps
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

// TestTanhFloat64IsAccurate measures tanhFloat64's distance from the exact
// value in float64 steps over the inputs it is responsible for, so the
// tanhSettleSteps margin rests on a measurement instead of an estimate.
func TestTanhFloat64IsAccurate(t *testing.T) {
	const margin = float64(tanhSettleSteps) / 2
	r := rand.New(rand.NewSource(41))
	var inputs []float32

	for i := 0; i < 20000; i++ {
		e := r.Intn(17) - 13
		x := float32(math.Ldexp(1+r.Float64(), e))
		if x < float32(math.Ldexp(1, -13)) || x >= 9.5 {
			continue
		}
		inputs = append(inputs, x)
	}
	for i := uint32(0); i < 4096; i++ {
		inputs = append(inputs, math.Float32frombits(0x39000000+i))
	}
	for i := uint32(0); i < 4096; i++ {
		inputs = append(inputs, math.Float32frombits(math.Float32bits(0.17)+i))
	}

	maxSteps := 0.0
	for _, x := range inputs {
		reference := tanhOracleBig(new(big.Float).SetPrec(64).SetFloat64(float64(x)), 192)
		got := new(big.Float).SetPrec(256).SetFloat64(tanhFloat64(float64(x)))
		difference := new(big.Float).SetPrec(256).Abs(new(big.Float).SetPrec(256).Sub(got, reference))
		ulp := math.Ldexp(1, reference.MantExp(nil)-53)
		steps, _ := new(big.Float).SetPrec(256).Quo(difference, big.NewFloat(ulp)).Float64()
		if steps >= margin {
			t.Fatalf("input bits %#08x: error %.6g float64 steps, want < %g", math.Float32bits(x), steps, margin)
		}
		if steps > maxSteps {
			maxSteps = steps
		}
	}
	t.Logf("tanhFloat64 maximum error: %.6g float64 steps (margin %g)", maxSteps, margin)
}

// TestTanhHardCasesAreHardAndRight keeps tanhHardCases honest: every entry must
// be an input the fast path cannot decide on its own, and must hold exactly the
// result the high-precision fallback computes for it.
func TestTanhHardCasesAreHardAndRight(t *testing.T) {
	if len(tanhHardCases) == 0 {
		t.Logf("tanhHardCases is empty: no float32 input has come within %d float64 steps of a midpoint", tanhSettleSteps)
	}
	for a, want := range tanhHardCases {
		x := math.Float32frombits(a)
		if tanhFloat64Settles(tanhFloat64(float64(x))) {
			t.Errorf("input bits %#08x: tanhFloat64Settles = true, want false", a)
			continue
		}
		if got := math.Float32bits(tanhOracle(x)); got != want {
			t.Errorf("input bits %#08x: table result bits %#08x, want %#08x", a, got, want)
		}
	}
}
