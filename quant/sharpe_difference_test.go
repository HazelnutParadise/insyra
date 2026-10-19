package quant

import (
	"errors"
	"math"
	"math/rand"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// bivariateReturns draws n pairs of normal returns with the given means, a
// common standard deviation and correlation rho.
func bivariateReturns(rng *rand.Rand, n int, meanA, meanB, sd, rho float64) (a, b []any) {
	a, b = make([]any, n), make([]any, n)
	for i := 0; i < n; i++ {
		z1, z2 := rng.NormFloat64(), rng.NormFloat64()
		a[i] = meanA + sd*z1
		b[i] = meanB + sd*(rho*z1+math.Sqrt(1-rho*rho)*z2)
	}
	return a, b
}

// Under equal true Sharpe ratios the test should reject at its nominal rate.
// This checks the variance formula by simulation: a wrong theta shows up as a
// rejection rate far from 5%.
func TestSharpeDifferenceTestHasItsNominalSize(t *testing.T) {
	rng := rand.New(rand.NewSource(20261018))
	const reps = 4000
	rejections := 0
	for r := 0; r < reps; r++ {
		a, b := bivariateReturns(rng, 250, 0.05, 0.05, 1, 0.5)
		res, err := SharpeDifferenceTest(insyra.NewDataList(a...), insyra.NewDataList(b...), 0)
		if err != nil {
			t.Fatal(err)
		}
		if res.PValue < 0.05 {
			rejections++
		}
	}
	rate := float64(rejections) / reps
	if rate < 0.037 || rate > 0.064 {
		t.Errorf("rejection rate under the null is %.4f, want about 0.05", rate)
	}
}

func TestSharpeDifferenceTestDetectsADifference(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	a, b := bivariateReturns(rng, 500, 0.15, 0.0, 1, 0.3)
	res, err := SharpeDifferenceTest(insyra.NewDataList(a...), insyra.NewDataList(b...), 0)
	if err != nil {
		t.Fatal(err)
	}
	if res.Difference <= 0 || res.PValue > 0.01 || res.PValueGreater > 0.005 || res.PValueLess < 0.995 {
		t.Errorf("got %+v", res)
	}
	swapped, _ := SharpeDifferenceTest(insyra.NewDataList(b...), insyra.NewDataList(a...), 0)
	if math.Abs(swapped.Z+res.Z) > 1e-12 || math.Abs(swapped.PValue-res.PValue) > 1e-12 {
		t.Errorf("swapping the series should flip Z only: %+v vs %+v", res, swapped)
	}
	sum := res.PValueGreater + res.PValueLess
	if math.Abs(sum-1) > 1e-12 {
		t.Errorf("one-tailed p-values add up to %v", sum)
	}
}

func TestSharpeDifferenceTestKnownValues(t *testing.T) {
	a := insyra.NewDataList(0.02, -0.01, 0.03, 0.01, -0.02, 0.04)
	b := insyra.NewDataList(0.01, 0.00, 0.02, -0.01, 0.01, 0.02)
	res, err := SharpeDifferenceTest(a, b, 0)
	if err != nil {
		t.Fatal(err)
	}
	// Per-period Sharpe with the n divisor: mean / sd.
	if math.Abs(res.SharpeA-0.5516772843673704) > 1e-12 || math.Abs(res.SharpeB-0.7808688094430303) > 1e-12 {
		t.Errorf("Sharpe ratios: %+v", res)
	}
	same, err := SharpeDifferenceTest(a, a, 0)
	if err != nil || same.Difference != 0 || same.Z != 0 || same.PValue != 1 {
		t.Errorf("identical series: %+v, %v", same, err)
	}
}

func TestSharpeDifferenceTestRefusesWhatItCannotTest(t *testing.T) {
	good := insyra.NewDataList(0.01, 0.02, -0.01, 0.03)
	flat := insyra.NewDataList(0.0, 0.0, 0.0, 0.0)
	if _, err := SharpeDifferenceTest(flat, good, 0); !errors.Is(err, ErrSharpeUndefined) {
		t.Errorf("zero-volatility a: got %v", err)
	}
	if _, err := SharpeDifferenceTest(good, flat, 0); !errors.Is(err, ErrSharpeUndefined) {
		t.Errorf("zero-volatility b: got %v", err)
	}
	var nilList *insyra.DataList
	cases := map[string]func() error{
		"nil": func() error { _, e := SharpeDifferenceTest(nilList, good, 0); return e },
		"NaN": func() error {
			_, e := SharpeDifferenceTest(insyra.NewDataList(0.1, math.NaN(), 0.2), good, 0)
			return e
		},
		"length": func() error { _, e := SharpeDifferenceTest(good, insyra.NewDataList(0.1, 0.2), 0); return e },
		"too few": func() error {
			_, e := SharpeDifferenceTest(insyra.NewDataList(0.1, 0.2), insyra.NewDataList(0.3, 0.1), 0)
			return e
		},
		"riskfree": func() error { _, e := SharpeDifferenceTest(good, good, math.NaN()); return e },
	}
	for name, f := range cases {
		if f() == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}
