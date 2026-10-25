package nn

import (
	"math"
	"math/big"
	"math/rand"
	"testing"
)

const log1pOraclePrec = 700

// log1pOracle is log(1+x) from the Mercator series on a reduced argument, a
// route that shares no code with the implementation's atanh series.
func log1pOracle(x float32) float32 {
	prec := uint(log1pOraclePrec)
	one := new(big.Float).SetPrec(prec).SetInt64(1)
	v := new(big.Float).SetPrec(prec).SetFloat64(float64(x))
	v.Add(v, one)
	m := new(big.Float).SetPrec(prec)
	e := v.MantExp(m) // v = m * 2^e, m in [0.5, 1)
	if m.Cmp(new(big.Float).SetPrec(prec).SetFloat64(0.75)) < 0 {
		m.Mul(m, new(big.Float).SetPrec(prec).SetInt64(2))
		e--
	}
	y := new(big.Float).SetPrec(prec).Sub(m, one)
	sum := new(big.Float).SetPrec(prec)
	pow := new(big.Float).SetPrec(prec).Set(y)
	for n := int64(1); n < 4000; n++ {
		term := new(big.Float).SetPrec(prec).Quo(pow, new(big.Float).SetPrec(prec).SetInt64(n))
		if n%2 == 0 {
			term.Neg(term)
		}
		next := new(big.Float).SetPrec(prec).Add(sum, term)
		if next.Cmp(sum) == 0 {
			break
		}
		sum = next
		pow.Mul(pow, y)
	}
	sum.Add(sum, new(big.Float).SetPrec(prec).Mul(expLogLn2Big(prec), new(big.Float).SetPrec(prec).SetInt64(int64(e))))
	out, _ := sum.Float32()
	return out
}

// expm1Oracle is e^x - 1 from the Taylor series without argument reduction,
// at a precision that covers the cancellation.
func expm1Oracle(x float32) float32 {
	prec := uint(log1pOraclePrec)
	xb := new(big.Float).SetPrec(prec).SetFloat64(float64(x))
	term := new(big.Float).SetPrec(prec).Set(xb)
	sum := new(big.Float).SetPrec(prec).Set(xb)
	for n := int64(2); n < 6000; n++ {
		term.Mul(term, xb)
		term.Quo(term, new(big.Float).SetPrec(prec).SetInt64(n))
		next := new(big.Float).SetPrec(prec).Add(sum, term)
		if next.Cmp(sum) == 0 {
			break
		}
		sum = next
	}
	out, _ := sum.Float32()
	return out
}

func sameFloat32Bits(a, b float32) bool {
	if math.IsNaN(float64(a)) && math.IsNaN(float64(b)) {
		return true
	}
	return math.Float32bits(a) == math.Float32bits(b)
}

func TestLog1pFloat32SpecialValues(t *testing.T) {
	inf := float32(math.Inf(1))
	negZero := float32(math.Copysign(0, -1))
	cases := []struct{ x, want float32 }{
		{float32(math.NaN()), float32(math.NaN())},
		{-2, float32(math.NaN())},
		{-1, -inf},
		{inf, inf},
		{0, 0},
		{negZero, negZero},
		{math.SmallestNonzeroFloat32, math.SmallestNonzeroFloat32},
		{math.MaxFloat32, 88.72284},
	}
	for _, c := range cases {
		if got := log1pFloat32(c.x); !sameFloat32Bits(got, c.want) && !(c.x == math.MaxFloat32 && got == log1pOracle(c.x)) {
			t.Errorf("log1p(%g) = %g, want %g", c.x, got, c.want)
		}
	}
}

func TestExpm1Float32SpecialValues(t *testing.T) {
	inf := float32(math.Inf(1))
	negZero := float32(math.Copysign(0, -1))
	cases := []struct{ x, want float32 }{
		{float32(math.NaN()), float32(math.NaN())},
		{inf, inf},
		{-inf, -1},
		{0, 0},
		{negZero, negZero},
		{90, inf},
		{-200, -1},
		{math.SmallestNonzeroFloat32, math.SmallestNonzeroFloat32},
	}
	for _, c := range cases {
		if got := expm1Float32(c.x); !sameFloat32Bits(got, c.want) {
			t.Errorf("expm1(%g) = %g, want %g", c.x, got, c.want)
		}
	}
}

func TestLog1pAndExpm1AreCorrectlyRoundedOnSamples(t *testing.T) {
	rng := rand.New(rand.NewSource(77))
	check := func(x float32) {
		if math.IsNaN(float64(x)) || math.IsInf(float64(x), 0) {
			return
		}
		if x > -1 {
			if got, want := log1pFloat32(x), log1pOracle(x); !sameFloat32Bits(got, want) {
				t.Fatalf("log1p(%g) = %g (%#x), want %g (%#x)", x, got, math.Float32bits(got), want, math.Float32bits(want))
			}
		}
		if x < 89 && x > -104 {
			if got, want := expm1Float32(x), expm1Oracle(x); !sameFloat32Bits(got, want) {
				t.Fatalf("expm1(%g) = %g (%#x), want %g (%#x)", x, got, math.Float32bits(got), want, math.Float32bits(want))
			}
		}
	}
	for i := 0; i < 4000; i++ {
		check(math.Float32frombits(rng.Uint32()))
	}
	for i := 0; i < 4000; i++ {
		check(float32(rng.NormFloat64() * 3))
		check(float32(math.Ldexp(rng.Float64()-0.5, -rng.Intn(60))))
	}
	for _, x := range []float32{1e-45, 1e-38, 1e-30, 5.9604645e-8, 1.1920929e-7, 0.5, 1, -0.5, -0.9999999, 16, 88, -17, -17.3, -88, math.MaxFloat32 / 4} {
		check(x)
	}
}

func TestLog1pTinyArgumentKeepsTheArgument(t *testing.T) {
	// log1p(x) = x - x^2/2 + ... rounds to x for |x| below about 2^-24, which
	// log(1+x) cannot see because 1+x rounds to 1.
	for _, x := range []float32{1e-10, -1e-10, 3e-8, -3e-8} {
		if got := log1pFloat32(x); math.Abs(float64(got)-float64(x)) > 1e-6*math.Abs(float64(x)) || got == 0 {
			t.Errorf("log1p(%g) = %g", x, got)
		}
	}
}
