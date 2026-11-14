package wgpu

import (
	"math"
	"math/big"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
)

// fastTanhExactScaled is tanh|x| * 2^58 at 600 bits.
func fastTanhExactScaled(ax uint32) *big.Float {
	z := new(big.Float).SetPrec(oraclePrec).Mul(bigF(math.Float32frombits(ax)), new(big.Float).SetPrec(oraclePrec).SetInt64(-2))
	term := new(big.Float).SetPrec(oraclePrec).SetInt64(1)
	e := new(big.Float).SetPrec(oraclePrec).SetInt64(1)
	for n := int64(1); n < 3000; n++ {
		term.Mul(term, z)
		term.Quo(term, new(big.Float).SetPrec(oraclePrec).SetInt64(n))
		next := new(big.Float).SetPrec(oraclePrec).Add(e, term)
		if next.Cmp(e) == 0 {
			break
		}
		e.Set(next)
	}
	one := new(big.Float).SetPrec(oraclePrec).SetInt64(1)
	q := new(big.Float).SetPrec(oraclePrec).Quo(new(big.Float).SetPrec(oraclePrec).Sub(one, e), new(big.Float).SetPrec(oraclePrec).Add(one, e))
	return q.SetMantExp(q, fastTanhFrac)
}

// fastTanhStart is the reciprocal start the host takes, moved by rel.
func fastTanhStart(ax uint32, rel float64) uint64 {
	t := math.Tanh(float64(math.Float32frombits(ax)))
	e := (1 - t) / (1 + t) // e^-2|x|
	return uint64(math.Ldexp(1/(1+e)*(1+rel), fastTanhFrac))
}

// TestFastTanhModelErrorBound measures the fast path's error against a
// 600-bit oracle on inputs across its range, from three reciprocal starts,
// and requires it far below fastTanhBound.
func TestFastTanhModelErrorBound(t *testing.T) {
	fixedLibrary()
	rng := rand.New(rand.NewSource(29))
	worst := 0.0
	var inputs []uint32
	for i := uint32(0); i < 300; i++ {
		inputs = append(inputs, 0x39000000+i, 0x41180000-1-i)
	}
	for i := 0; i < 3000; i++ {
		inputs = append(inputs, uint32(0x39000000+rng.Int63n(0x41180000-0x39000000)))
	}
	for _, ax := range inputs {
		exact := fastTanhExactScaled(ax)
		for _, rel := range []float64{0, 1.0 / (1 << 18), -1.0 / (1 << 18)} {
			got := fastTanhModel(ax, fixedWords, fastTanhStart(ax, rel))
			diff, _ := new(big.Float).Sub(exact, new(big.Float).SetUint64(got)).Float64()
			worst = math.Max(worst, math.Abs(diff))
		}
	}
	t.Logf("largest error %.2f units of 2^-58 over %d inputs", worst, len(inputs))
	if worst > fastTanhBound/8 {
		t.Fatalf("largest error %.2f units is within 8 times of the bound %d", worst, fastTanhBound)
	}
}

// TestFastTanhModelFallsBackRarely runs the host model over every positive
// input of its range and counts the ones it sends to the 128-bit path.
func TestFastTanhModelFallsBackRarely(t *testing.T) {
	if testing.Short() {
		t.Skip("135 million inputs")
	}
	fixedLibrary()
	var fallbacks atomic.Int64
	workers := runtime.NumCPU()
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w uint32) {
			defer wg.Done()
			var f int64
			for ax := 0x39000000 + w; ax < 0x41180000; ax += uint32(workers) {
				if _, ok := fastTanhSettle(fastTanhModel(ax, fixedWords, fastTanhStart(ax, 0))); !ok {
					f++
				}
			}
			fallbacks.Add(f)
		}(uint32(w))
	}
	wg.Wait()
	t.Logf("%d of %d inputs fall back", fallbacks.Load(), 0x41180000-0x39000000)
	if fallbacks.Load() > 1000 {
		t.Fatalf("%d inputs fall back, expected a few hundred", fallbacks.Load())
	}
}
