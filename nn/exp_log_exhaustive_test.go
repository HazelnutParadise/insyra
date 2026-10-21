package nn

import (
	"fmt"
	"math"
	"math/big"
	"os"
	"runtime"
	"sort"
	"sync"
	"testing"
)

const (
	expFloat64GoldenHash uint64 = 0x08bd64736db2de1d
	logFloat64GoldenHash uint64 = 0x4de33c339e7c9e0a
)

func expLogMix64(z uint64) uint64 {
	z ^= z >> 30
	z *= 0xbf58476d1ce4e5b9
	z ^= z >> 27
	z *= 0x94d049bb133111eb
	z ^= z >> 31
	return z
}

func TestExpLogFloat64IsTheSameEverywhere(t *testing.T) {
	requireExpLogExhaustive(t)
	expHash := walkExpRange(t, false).hash
	logHash := walkLogRange(t, false).hash
	if expHash != expFloat64GoldenHash {
		t.Errorf("%s/%s expFloat64 hash = %#016x, want %#016x", runtime.GOOS, runtime.GOARCH, expHash, expFloat64GoldenHash)
	}
	if logHash != logFloat64GoldenHash {
		t.Errorf("%s/%s logFloat64 hash = %#016x, want %#016x", runtime.GOOS, runtime.GOARCH, logHash, logFloat64GoldenHash)
	}
	t.Logf("%s/%s exp hash %#016x; log hash %#016x", runtime.GOOS, runtime.GOARCH, expHash, logHash)
}

func TestExpLogCorrectlyRoundedExhaustive(t *testing.T) {
	requireExpLogExhaustive(t)
	expResult := walkExpRange(t, true)
	logResult := walkLogRange(t, true)
	checkExpLogCases(t, "exp", expResult, 0)
	checkExpLogCases(t, "log", logResult, 1)
	for _, mismatch := range expResult.mismatches {
		t.Error(mismatch)
	}
	for _, mismatch := range logResult.mismatches {
		t.Error(mismatch)
	}
	if expResult.maxErr >= float64(expLogSettleSteps)/2 {
		t.Errorf("expFloat64 maximum measured error %.8g float64 steps, want < %g", expResult.maxErr, float64(expLogSettleSteps)/2)
	}
	if logResult.maxErr >= float64(expLogSettleSteps)/2 {
		t.Errorf("logFloat64 maximum measured error %.8g float64 steps, want < %g", logResult.maxErr, float64(expLogSettleSteps)/2)
	}
	t.Logf("exp: %d inputs, %d undecided, max error %.8g float64 steps", expResult.total, len(expResult.cases), expResult.maxErr)
	t.Logf("log: %d inputs, %d undecided, max error %.8g float64 steps", logResult.total, len(logResult.cases), logResult.maxErr)
}

func requireExpLogExhaustive(t *testing.T) {
	t.Helper()
	if os.Getenv("INSYRA_NN_EXHAUSTIVE") != "1" {
		t.Skip("set INSYRA_NN_EXHAUSTIVE=1 to enumerate all exp/log float32 inputs")
	}
}

type expLogCase struct {
	input uint32
	value uint32
}

type expLogWalk struct {
	hash       uint64
	total      uint64
	cases      []expLogCase
	maxErr     float64
	mismatches []string
}

func walkExpRange(t *testing.T, oracle bool) expLogWalk {
	t.Helper()
	// In monotonically ordered IEEE bits, the negative half runs backwards.
	return mergeExpWalks(
		walkExpBits(t, 0xc2d00000, uint64(0xc2d00000-0x80000000)+1, -1, oracle),
		walkExpBits(t, 0, uint64(0x42b20000)+1, 1, oracle),
	)
}

func walkExpBits(t *testing.T, start uint32, total uint64, direction int, oracle bool) expLogWalk {
	t.Helper()
	return parallelExpLogWalk(t, total, func(i uint64) uint32 {
		if direction < 0 {
			return uint32(uint64(start) - i)
		}
		return uint32(uint64(start) + i)
	}, func(bits uint32) (float64, bool, uint32, *big.Float, uint32) {
		x := math.Float32frombits(bits)
		approx := expFloat64(float64(x))
		if expLogFloat64Settles(approx) {
			return approx, false, 0, nil, 0
		}
		if !oracle {
			return approx, true, 0, nil, 0
		}
		exact := expHighPrecisionExact(x)
		want, _ := exact.Float32()
		return approx, true, math.Float32bits(want), exact, math.Float32bits(expFloat32(x))
	})
}

func walkLogRange(t *testing.T, oracle bool) expLogWalk {
	t.Helper()
	return parallelExpLogWalk(t, 0x7f7fffff, func(i uint64) uint32 { return uint32(i + 1) }, func(bits uint32) (float64, bool, uint32, *big.Float, uint32) {
		x := math.Float32frombits(bits)
		approx := logFloat64(float64(x))
		if expLogFloat64Settles(approx) {
			return approx, false, 0, nil, 0
		}
		if !oracle {
			return approx, true, 0, nil, 0
		}
		exact := logHighPrecisionExact(x)
		want, _ := exact.Float32()
		return approx, true, math.Float32bits(want), exact, math.Float32bits(logFloat32(x))
	})
}

func parallelExpLogWalk(t *testing.T, total uint64, bitsAt func(uint64) uint32, calculate func(uint32) (float64, bool, uint32, *big.Float, uint32)) expLogWalk {
	t.Helper()
	workers := runtime.NumCPU()
	if workers > 64 {
		workers = 64
	}
	chunk, remainder := total/uint64(workers), total%uint64(workers)
	parts := make([]expLogWalk, workers)
	var wg sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		start := uint64(worker) * chunk
		end := start + chunk
		if worker == workers-1 {
			end += remainder
		}
		wg.Add(1)
		go func(worker int, start, end uint64) {
			defer wg.Done()
			part := expLogWalk{total: end - start}
			for i := start; i < end; i++ {
				bits := bitsAt(i)
				approx, undecided, expected, exact, actual := calculate(bits)
				part.hash += expLogMix64(uint64(bits)<<32 ^ math.Float64bits(approx))
				if !undecided {
					continue
				}
				part.cases = append(part.cases, expLogCase{bits, expected})
				if exact != nil && actual != expected {
					part.mismatches = append(part.mismatches, fmt.Sprintf("input bits %#08x: result bits %#08x, want %#08x", bits, actual, expected))
				}
				if exact != nil {
					part.maxErr = math.Max(part.maxErr, expLogErrorSteps(approx, exact))
				}
			}
			parts[worker] = part
		}(worker, start, end)
	}
	wg.Wait()
	var result expLogWalk
	for _, part := range parts {
		result.hash += part.hash
		result.total += part.total
		result.maxErr = math.Max(result.maxErr, part.maxErr)
		result.cases = append(result.cases, part.cases...)
		result.mismatches = append(result.mismatches, part.mismatches...)
	}
	sort.Slice(result.cases, func(i, j int) bool { return result.cases[i].input < result.cases[j].input })
	return result
}

func mergeExpWalks(a, b expLogWalk) expLogWalk {
	a.hash += b.hash
	a.total += b.total
	if b.maxErr > a.maxErr {
		a.maxErr = b.maxErr
	}
	a.cases = append(a.cases, b.cases...)
	a.mismatches = append(a.mismatches, b.mismatches...)
	sort.Slice(a.cases, func(i, j int) bool { return a.cases[i].input < a.cases[j].input })
	return a
}

func expLogErrorSteps(approx float64, exact *big.Float) float64 {
	if exact.IsInf() {
		return 0
	}
	approxBig := new(big.Float).SetPrec(exact.Prec()).SetFloat64(approx)
	difference := new(big.Float).SetPrec(exact.Prec()).Abs(new(big.Float).SetPrec(exact.Prec()).Sub(approxBig, exact))
	if difference.Sign() == 0 {
		return 0
	}
	e := exact.MantExp(nil)
	ulp := new(big.Float).SetPrec(exact.Prec()).SetMantExp(new(big.Float).SetPrec(exact.Prec()).SetInt64(1), e-53)
	steps := new(big.Float).SetPrec(exact.Prec()).Quo(difference, ulp)
	value, _ := steps.Float64()
	return value
}

func checkExpLogCases(t *testing.T, name string, result expLogWalk, tag uint32) {
	t.Helper()
	want := make(map[uint64]uint32, len(result.cases))
	for _, c := range result.cases {
		want[expLogKey(tag, math.Float32frombits(c.input))] = c.value
	}
	var missing, stale []string
	for key, value := range want {
		if got, ok := expLogHardCases[key]; !ok || got != value {
			missing = append(missing, fmt.Sprintf("0x%08x: 0x%08x", key, value))
		}
	}
	for key := range expLogHardCases {
		if uint32(key>>32) == tag {
			if _, ok := want[key]; !ok {
				stale = append(stale, fmt.Sprintf("0x%08x", key))
			}
		}
	}
	if len(missing) != 0 || len(stale) != 0 {
		sort.Strings(missing)
		sort.Strings(stale)
		t.Errorf("%s hard cases differ: missing or wrong=%v stale=%v; paste this table content into exp_log_hard_cases.go:\n%s", name, missing, stale, formatExpLogCases(name, result, tag))
	}
}

func formatExpLogCases(name string, result expLogWalk, tag uint32) string {
	var out string
	for _, c := range result.cases {
		key := expLogKey(tag, math.Float32frombits(c.input))
		out += fmt.Sprintf("\t0x%016x: 0x%08x, // %s input bits 0x%08x\n", key, c.value, name, c.input)
	}
	return out
}
