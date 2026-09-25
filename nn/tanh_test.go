package nn

import (
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"runtime"
	"sort"
	"sync"
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
		t.Fatal("tanhHardCases is empty, but the exhaustive run lists four inputs the fast path cannot decide")
	}
	for a, want := range tanhHardCases {
		x := math.Float32frombits(a)
		if tanhFloat64Settles(tanhFloat64(float64(x))) {
			t.Errorf("input bits %#08x: tanhFloat64Settles = true, want false", a)
			continue
		}
		if got := math.Float32bits(tanhOracle(x)); got != want {
			t.Errorf("input bits %#08x: oracle result bits %#08x, table holds %#08x", a, got, want)
		}
		if got := math.Float32bits(tanhFloat32(x)); got != want {
			t.Errorf("input bits %#08x: tanhFloat32 result bits %#08x, want %#08x", a, got, want)
		}
		if got, negated := math.Float32bits(tanhFloat32(-x)), want|0x80000000; got != negated {
			t.Errorf("input bits %#08x negated: tanhFloat32 result bits %#08x, want %#08x", a, got, negated)
		}
	}
}

// tanhFloat64GoldenHash is TestTanhFloat64IsTheSameEverywhere's hash of
// tanhFloat64 over every positive float32 in [2^-13, 9.5), computed on the
// darwin/arm64 machine where TestTanhIsCorrectlyRoundedExhaustive measured its
// error. A platform that computes another value computes different bits.
const tanhFloat64GoldenHash uint64 = 0x9b8d83bd1588cfdd

func tanhMix64(z uint64) uint64 {
	z ^= z >> 30
	z *= 0xbf58476d1ce4e5b9
	z ^= z >> 27
	z *= 0x94d049bb133111eb
	z ^= z >> 31
	return z
}

// TestTanhFloat64IsTheSameEverywhere walks every input tanhFloat64 is
// responsible for and pins what it computes two ways: the undecided inputs have
// to be exactly tanhHardCases' keys, and every result's bits have to hash to
// tanhFloat64GoldenHash. The exhaustive comparison is what measured the error
// tanhSettleSteps rests on, and it does not run in CI, so without this a
// platform computing other bits, or a change moving a result into or out of
// tanhSettleSteps of a float32 midpoint, would not be caught until someone
// reran the exhaustive run.
func TestTanhFloat64IsTheSameEverywhere(t *testing.T) {
	if testing.Short() {
		t.Skip("walks all 135790592 positive float32 inputs in [2^-13, 9.5)")
	}

	// 0x39000000 is 2^-13 and 0x41180000 is 9.5, so the bit patterns in between
	// are exactly the positive float32 inputs tanhFloat32 hands tanhFloat64.
	const firstBits = uint32(0x39000000)
	const lastBits = uint32(0x41180000)
	const total = uint64(lastBits - firstBits)

	workers := runtime.NumCPU()
	chunk := total / uint64(workers)
	remainder := total % uint64(workers)
	hashes := make([]uint64, workers)
	changedByWorker := make([]uint64, workers)
	undecidedByWorker := make([][]uint32, workers)

	var wg sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		start := uint64(firstBits) + uint64(worker)*chunk
		end := start + chunk
		if worker == workers-1 {
			end += remainder
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Every worker keeps its own accumulator, its own count and its own
			// undecided list, so the loop below never touches shared state and
			// the result does not depend on how the ranges were split.
			var hash, changes uint64
			var undecided []uint32
			for a := start; a < end; a++ {
				x := math.Float32frombits(uint32(a))
				value := tanhFloat64(float64(x))
				hash += tanhMix64(a<<32 ^ math.Float64bits(value))
				if !tanhFloat64Settles(value) {
					undecided = append(undecided, uint32(a))
				}
				if math.Float32bits(float32(math.Tanh(float64(x)))) != math.Float32bits(tanhFloat32(x)) {
					changes++
				}
			}
			hashes[worker] = hash
			changedByWorker[worker] = changes
			undecidedByWorker[worker] = undecided
		}()
	}
	wg.Wait()

	var hash, changed uint64
	var undecided []uint32
	for _, h := range hashes {
		hash += h
	}
	for _, c := range changedByWorker {
		changed += c
	}
	for _, list := range undecidedByWorker {
		undecided = append(undecided, list...)
	}
	sort.Slice(undecided, func(i, j int) bool { return undecided[i] < undecided[j] })

	// tanhHardCases exists to answer exactly the inputs the fast path cannot
	// decide, so the two sets have to match one for one in both directions.
	keys := make([]uint32, 0, len(tanhHardCases))
	for key := range tanhHardCases {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	var unlisted, noLongerUndecided []string
	i, j := 0, 0
	for i < len(undecided) || j < len(keys) {
		switch {
		case j == len(keys) || (i < len(undecided) && undecided[i] < keys[j]):
			unlisted = append(unlisted, fmt.Sprintf("%#08x", undecided[i]))
			i++
		case i == len(undecided) || keys[j] < undecided[i]:
			noLongerUndecided = append(noLongerUndecided, fmt.Sprintf("%#08x", keys[j]))
			j++
		default:
			i++
			j++
		}
	}
	if len(unlisted) > 0 || len(noLongerUndecided) > 0 {
		t.Errorf("undecided but not in tanhHardCases: %v; in tanhHardCases but no longer undecided: %v; the inputs within %d float64 steps of a float32 midpoint changed; rerun INSYRA_EXHAUSTIVE_TESTS=1 go test -run TestTanhIsCorrectlyRoundedExhaustive ./nn/ and update tanhHardCases", unlisted, noLongerUndecided, tanhSettleSteps)
	}

	if hash != tanhFloat64GoldenHash {
		t.Errorf("%s/%s: tanhFloat64's hash over %d inputs is %#016x, want %#016x: it computes different bits here than on the machine where its error was measured, so that measurement and tanhHardCases do not cover this platform; rerun the exhaustive comparison here", runtime.GOOS, runtime.GOARCH, total, hash, tanhFloat64GoldenHash)
	}

	// The last number counts the results that differ from float32(math.Tanh(x)),
	// the tanh this replaced. math.Tanh's bits vary by platform, so it records
	// how many middle-range results the change moved on the platform running it.
	t.Logf("%s/%s: %d inputs, hash %#016x, %d undecided, %d of them change from float32(math.Tanh(x))", runtime.GOOS, runtime.GOARCH, total, hash, len(undecided), changed)
}
