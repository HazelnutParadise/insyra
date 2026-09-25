package nn

import (
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
)

func tanhOracleBig(a *big.Float, prec uint) *big.Float {
	if a.Sign() == 0 {
		return new(big.Float).SetPrec(prec)
	}

	z := new(big.Float).SetPrec(prec+96).Mul(a, big.NewFloat(2))
	z64, _ := z.Float64()
	extra := uint(math.Ceil(2 * z64 * math.Log2E))
	workPrec := prec + 96 + extra
	z.SetPrec(workPrec)
	z.Mul(a, big.NewFloat(2))
	negativeZ := new(big.Float).SetPrec(workPrec).Neg(z)
	term := new(big.Float).SetPrec(workPrec).SetInt64(1)
	sum := new(big.Float).SetPrec(workPrec).SetInt64(1)
	threshold := new(big.Float).SetPrec(workPrec).SetMantExp(new(big.Float).SetInt64(1), -int(workPrec))
	for k := 1; ; k++ {
		term.Mul(term, negativeZ)
		term.Quo(term, new(big.Float).SetPrec(workPrec).SetInt64(int64(k)))
		sum.Add(sum, term)
		if new(big.Float).SetPrec(workPrec).Abs(term).Cmp(threshold) < 0 {
			break
		}
	}

	one := new(big.Float).SetPrec(workPrec).SetInt64(1)
	numerator := new(big.Float).SetPrec(workPrec).Sub(one, sum)
	denominator := new(big.Float).SetPrec(workPrec).Add(one, sum)
	return new(big.Float).SetPrec(prec).Quo(numerator, denominator)
}

func tanhOracle(x float32) float32 {
	if math.IsNaN(float64(x)) {
		return x
	}
	if math.IsInf(float64(x), 0) {
		return float32(math.Copysign(1, float64(x)))
	}
	if x == 0 {
		return x
	}

	abs := x
	if abs < 0 {
		abs = -abs
	}
	a := new(big.Float).SetPrec(64).SetFloat64(float64(abs))
	prec := uint(192)
	for prec <= 4096 {
		y := tanhOracleBig(a, prec)
		f, _ := y.Float32()
		previous := math.Nextafter32(f, float32(math.Inf(-1)))
		next := math.Nextafter32(f, float32(math.Inf(1)))
		fBig := new(big.Float).SetPrec(prec).SetFloat64(float64(f))
		previousBig := new(big.Float).SetPrec(prec).SetFloat64(float64(previous))
		nextBig := new(big.Float).SetPrec(prec).SetFloat64(float64(next))
		lowerSum := new(big.Float).SetPrec(prec).Add(fBig, previousBig)
		upperSum := new(big.Float).SetPrec(prec).Add(fBig, nextBig)
		lowerMidpoint := new(big.Float).SetPrec(prec).Quo(lowerSum, big.NewFloat(2))
		upperMidpoint := new(big.Float).SetPrec(prec).Quo(upperSum, big.NewFloat(2))
		lowerDistance := new(big.Float).SetPrec(prec).Abs(new(big.Float).SetPrec(prec).Sub(y, lowerMidpoint))
		upperDistance := new(big.Float).SetPrec(prec).Abs(new(big.Float).SetPrec(prec).Sub(y, upperMidpoint))
		scale := new(big.Float).SetPrec(prec).SetMantExp(new(big.Float).SetInt64(1), -int(prec-8))
		margin := new(big.Float).SetPrec(prec).Mul(y, scale)
		if lowerDistance.Cmp(margin) > 0 && upperDistance.Cmp(margin) > 0 {
			return float32(math.Copysign(float64(f), float64(x)))
		}
		prec *= 2
	}
	panic("tanhOracle precision exceeded 4096 bits")
}

func TestTanhOracleAnchors(t *testing.T) {
	tests := []struct {
		x     float32
		truth string
	}{
		{x: 0.5, truth: "0.462117157260009758502318483643672548730289280330113038552732"},
		{x: 1, truth: "0.761594155955764888119458282604793590412768597257936551596811"},
		{x: 2, truth: "0.964027580075816883946413724100923150255029976240934776048263"},
		{x: -3, truth: "-0.995054753686730451331880185255488475097813854700282491823877"},
		{x: 9.25, truth: "0.999999981525100846719761142643031558775543790583967578363828"},
	}
	limit, _, err := big.ParseFloat("1e-50", 10, 512, big.ToNearestEven)
	if err != nil {
		t.Fatalf("parse limit: %v", err)
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("%g", tc.x), func(t *testing.T) {
			a := new(big.Float).SetPrec(64).SetFloat64(math.Abs(float64(tc.x)))
			got := tanhOracleBig(a, 256)
			if tc.x < 0 {
				got.Neg(got)
			}
			truth, _, err := big.ParseFloat(tc.truth, 10, 256, big.ToNearestEven)
			if err != nil {
				t.Fatalf("parse truth: %v", err)
			}
			difference := new(big.Float).SetPrec(512).Abs(new(big.Float).SetPrec(512).Sub(got, truth))
			if difference.Cmp(limit) >= 0 {
				t.Fatalf("absolute difference = %s, want < 1e-50", difference.Text('g', 20))
			}
			want, _ := truth.Float32()
			if bits := math.Float32bits(tanhOracle(tc.x)); bits != math.Float32bits(want) {
				t.Fatalf("float32 bits = %#08x, want %#08x", bits, math.Float32bits(want))
			}
		})
	}
}

func TestTanhIsCorrectlyRoundedExhaustive(t *testing.T) {
	if os.Getenv("INSYRA_EXHAUSTIVE_TESTS") != "1" {
		t.Skip("set INSYRA_EXHAUSTIVE_TESTS=1 to compare all 2^32 float32 inputs")
	}

	type rangeStats struct {
		inputs     atomic.Uint64
		mismatches atomic.Uint64
	}
	type mismatchExample struct {
		input    uint32
		actual   uint32
		expected uint32
	}

	var tiny, large, special, middle rangeStats
	examples := make([]mismatchExample, 0, 20)
	var examplesMu sync.Mutex
	var maxError float64
	var maxErrorMu sync.Mutex

	const totalBits = uint64(1) << 32
	workers := runtime.NumCPU()
	chunk := totalBits / uint64(workers)
	remainder := totalBits % uint64(workers)
	tinyLimit := float32(math.Ldexp(1, -13))
	largeLimit := float32(9.5)

	var wg sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		start := uint64(worker) * chunk
		end := start + chunk
		if worker == workers-1 {
			end += remainder
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			localMaxError := 0.0
			for index := start; index < end; index++ {
				inputBits := uint32(index)
				x := math.Float32frombits(inputBits)
				actual := tanhFloat32(x)
				magnitude := x
				if magnitude < 0 {
					magnitude = -magnitude
				}

				var stats *rangeStats
				var expected float32
				mismatch := false
				middleInput := false
				switch {
				case math.IsNaN(float64(x)):
					stats = &special
					expected = x
					mismatch = !math.IsNaN(float64(actual))
				case math.IsInf(float64(x), 0):
					stats = &special
					expected = float32(math.Copysign(1, float64(x)))
					mismatch = math.Float32bits(actual) != math.Float32bits(expected)
				case magnitude < tinyLimit:
					stats = &tiny
					expected = x
					mismatch = math.Float32bits(actual) != math.Float32bits(expected)
				case magnitude >= largeLimit:
					stats = &large
					expected = float32(math.Copysign(1, float64(x)))
					mismatch = math.Float32bits(actual) != math.Float32bits(expected)
				default:
					stats = &middle
					expected = tanhOracle(x)
					mismatch = math.Float32bits(actual) != math.Float32bits(expected)
					middleInput = true
				}
				stats.inputs.Add(1)
				if mismatch {
					stats.mismatches.Add(1)
					examplesMu.Lock()
					if len(examples) < cap(examples) {
						examples = append(examples, mismatchExample{
							input:    inputBits,
							actual:   math.Float32bits(actual),
							expected: math.Float32bits(expected),
						})
					}
					examplesMu.Unlock()
				}

				if middleInput {
					a := new(big.Float).SetPrec(64).SetFloat64(float64(magnitude))
					reference := tanhOracleBig(a, 192)
					t := new(big.Float).SetPrec(256).SetFloat64(math.Abs(math.Tanh(float64(x))))
					diff := new(big.Float).SetPrec(256).Sub(t, reference)
					ulp := math.Ldexp(1, reference.MantExp(nil)-53)
					errorUlps, _ := new(big.Float).Quo(new(big.Float).Abs(diff), big.NewFloat(ulp)).Float64()
					if errorUlps > localMaxError {
						localMaxError = errorUlps
					}
				}
			}
			maxErrorMu.Lock()
			if localMaxError > maxError {
				maxError = localMaxError
			}
			maxErrorMu.Unlock()
		}()
	}
	wg.Wait()

	fmt.Printf("tiny: inputs=%d mismatches=%d\n", tiny.inputs.Load(), tiny.mismatches.Load())
	fmt.Printf("large: inputs=%d mismatches=%d\n", large.inputs.Load(), large.mismatches.Load())
	fmt.Printf("special: inputs=%d mismatches=%d\n", special.inputs.Load(), special.mismatches.Load())
	fmt.Printf("middle: inputs=%d mismatches=%d\n", middle.inputs.Load(), middle.mismatches.Load())
	if len(examples) == 0 {
		fmt.Println("mismatch examples: none")
	} else {
		fmt.Printf("mismatch examples: %d\n", len(examples))
		for _, example := range examples {
			fmt.Printf("  input=%#08x actual=%#08x expected=%#08x\n", example.input, example.actual, example.expected)
		}
	}
	fmt.Printf("math.Tanh maximum middle error: %.6g float64 ulp (against the exact value)\n", maxError)

	totalMismatches := tiny.mismatches.Load() + large.mismatches.Load() + special.mismatches.Load() + middle.mismatches.Load()
	if totalMismatches > 0 {
		t.Errorf("Tanh mismatches: total=%d tiny=%d large=%d special=%d middle=%d", totalMismatches, tiny.mismatches.Load(), large.mismatches.Load(), special.mismatches.Load(), middle.mismatches.Load())
	}
}

func TestTanhAnalyticRangesAgreeWithOracle(t *testing.T) {
	rng := rand.New(rand.NewSource(23))

	for i := 0; i < 20000; i++ {
		magnitude := math.Float32frombits(uint32(rng.Int63n(0x38ffffff) + 1))
		x := magnitude
		if rng.Intn(2) != 0 {
			x = -x
		}
		if got, want := math.Float32bits(tanhOracle(x)), math.Float32bits(x); got != want {
			t.Fatalf("tiny sample %d: input bits %#08x, oracle bits %#08x, want %#08x", i, math.Float32bits(x), got, want)
		}
	}

	for i := 0; i < 20000; i++ {
		x := float32(9.5 + rng.Float64()*(100-9.5))
		if rng.Intn(2) != 0 {
			x = -x
		}
		want := math.Float32bits(float32(math.Copysign(1, float64(x))))
		if got := math.Float32bits(tanhOracle(x)); got != want {
			t.Fatalf("large sample %d: input bits %#08x, oracle bits %#08x, want %#08x", i, math.Float32bits(x), got, want)
		}
	}
}
