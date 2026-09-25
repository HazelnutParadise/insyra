package nn

import (
	"math"
	"math/big"
	"sync"
)

var (
	ln2CacheMu sync.Mutex
	ln2Cache   = make(map[uint]*big.Float)
)

// tanhFloat32 returns the correctly rounded value of tanh for one float32
// input, using a certified float64 fast path and an independent fallback.
func tanhFloat32(x float32) float32 {
	if x != x {
		return x
	}
	// 0x39000000 is 2^-13, 0x41180000 is 9.5, and the comparison is on the
	// bit patterns so it also catches +-Inf in the second range.
	a := math.Float32bits(x) & 0x7fffffff
	if a < 0x39000000 {
		// For a = |x| < 2^-13, tanh(x) lies between x-x^3/3 and x. Since
		// |x^3/3| = a^3/3 < a*2^-25 because a^2 < 2^-26, the true value is
		// less than half the distance to the next float32 and rounds to x.
		return x
	}
	if a >= 0x41180000 {
		// For a >= 9.5, 1-tanh(a)=2/(e^(2a)+1)<2e^(-19)<2^(-25).
		// The latter is half the gap from 1 to its next float32, so the
		// correctly rounded result is 1 with the sign of x.
		return float32(math.Copysign(1, float64(x)))
	}

	t := math.Tanh(float64(x))
	if tanhFloat64Settles(t) {
		return float32(t)
	}
	return tanhHighPrecision(x)
}

// tanhFloat64Settles reports whether float32(t) is the correctly rounded tanh
// when t is math.Tanh's float64 result for an input with 2^-13 <= |x| < 9.5.
// There 2^-14 < |t| < 1, where every float32 is normal. Within one binade a
// float32 step is 2^29 float64 steps, so the float32 midpoints in t's binade
// are exactly the float64 values whose low 29 significand bits equal 1<<28,
// and the only midpoint outside it that could be near, the one just below the
// binade's lowest float32, is at least 2^27 float64 steps from any t in the
// binade. low - 1<<28 therefore counts the float64 steps from t to the nearest
// midpoint. math.Tanh stays within 1.31 float64 steps of the true value on
// every float32 input (TestTanhIsCorrectlyRoundedExhaustive; the Cephes source
// it is ported from reports a peak relative error of 2.5e-16), so when t is
// more than 2^13 steps from every midpoint the true value rounds to the same
// float32 as t. That is Ziv's method with an integer test.
func tanhFloat64Settles(t float64) bool {
	d := int64(math.Float64bits(t)&(1<<29-1)) - 1<<28
	return d > 1<<13 || d < -(1<<13)
}

// tanhHighPrecision evaluates tanh with a range-reduced big.Float exponential
// and increases precision until both float32 rounding boundaries are clear.
func tanhHighPrecision(x float32) float32 {
	a := x
	if a < 0 {
		a = -a
	}
	z := new(big.Float).SetPrec(64).SetFloat64(float64(a))
	z.Mul(z, big.NewFloat(2))

	for prec := uint(128); prec <= 4096; prec *= 2 {
		w := prec + 64
		z64, _ := z.Float64()
		k := int(math.Round(z64 / math.Ln2))
		ln2 := ln2Big(w)
		shift := new(big.Float).SetPrec(w).SetInt64(int64(k))
		shift.Mul(shift, ln2)
		r := new(big.Float).SetPrec(w).Sub(z, shift)
		e := expBig(r, w)
		e.SetMantExp(e, k)

		one := new(big.Float).SetPrec(w).SetInt64(1)
		two := new(big.Float).SetPrec(w).SetInt64(2)
		denominator := new(big.Float).SetPrec(w).Add(one, e)
		fraction := new(big.Float).SetPrec(w).Quo(two, denominator)
		y := new(big.Float).SetPrec(prec).Sub(new(big.Float).SetPrec(prec).SetInt64(1), fraction)

		f32, _ := y.Float32()
		prev := math.Nextafter32(f32, float32(math.Inf(-1)))
		next := math.Nextafter32(f32, float32(math.Inf(1)))
		fBig := new(big.Float).SetPrec(prec).SetFloat64(float64(f32))
		prevBig := new(big.Float).SetPrec(prec).SetFloat64(float64(prev))
		nextBig := new(big.Float).SetPrec(prec).SetFloat64(float64(next))
		twoAtPrec := new(big.Float).SetPrec(prec).SetInt64(2)
		lowerSum := new(big.Float).SetPrec(prec).Add(fBig, prevBig)
		upperSum := new(big.Float).SetPrec(prec).Add(fBig, nextBig)
		lowerMidpoint := new(big.Float).SetPrec(prec).Quo(lowerSum, twoAtPrec)
		upperMidpoint := new(big.Float).SetPrec(prec).Quo(upperSum, twoAtPrec)
		lowerDistance := new(big.Float).SetPrec(prec).Abs(new(big.Float).SetPrec(prec).Sub(y, lowerMidpoint))
		upperDistance := new(big.Float).SetPrec(prec).Abs(new(big.Float).SetPrec(prec).Sub(y, upperMidpoint))
		scale := new(big.Float).SetPrec(prec).SetMantExp(new(big.Float).SetPrec(prec).SetInt64(1), -int(prec-8))
		margin := new(big.Float).SetPrec(prec).Mul(y, scale)
		if lowerDistance.Cmp(margin) > 0 && upperDistance.Cmp(margin) > 0 {
			return float32(math.Copysign(float64(f32), float64(x)))
		}
	}
	panic("tanh of a nonzero float32 is transcendental, so it is never a float32 midpoint; this is unreachable")
}

// expBig evaluates exp with the Taylor series through the first term whose
// absolute value is below 2^-prec, keeping the calculation at prec bits.
func expBig(r *big.Float, prec uint) *big.Float {
	one := new(big.Float).SetPrec(prec).SetInt64(1)
	sum := new(big.Float).SetPrec(prec).Set(one)
	term := new(big.Float).SetPrec(prec).Set(one)
	threshold := new(big.Float).SetPrec(prec).SetMantExp(new(big.Float).SetPrec(prec).SetInt64(1), -int(prec))
	for n := uint(1); ; n++ {
		term.Mul(term, r)
		term.Quo(term, new(big.Float).SetPrec(prec).SetInt64(int64(n)))
		sum.Add(sum, term)
		if new(big.Float).SetPrec(prec).Abs(term).Cmp(threshold) < 0 {
			break
		}
	}
	return sum
}

// ln2Big computes ln(2) = 2*atanh(1/3), caching one immutable value per
// requested precision and returning a copy to keep callers independent.
func ln2Big(prec uint) *big.Float {
	ln2CacheMu.Lock()
	defer ln2CacheMu.Unlock()
	if cached, ok := ln2Cache[prec]; ok {
		return new(big.Float).Copy(cached)
	}

	one := new(big.Float).SetPrec(prec).SetInt64(1)
	three := new(big.Float).SetPrec(prec).SetInt64(3)
	term := new(big.Float).SetPrec(prec).Quo(one, three)
	sum := new(big.Float).SetPrec(prec).SetInt64(0)
	threshold := new(big.Float).SetPrec(prec).SetMantExp(new(big.Float).SetPrec(prec).SetInt64(1), -int(prec+8))
	for k := uint(0); ; k++ {
		sum.Add(sum, term)
		if new(big.Float).SetPrec(prec).Abs(term).Cmp(threshold) < 0 {
			break
		}
		numerator := int64(2*k + 1)
		denominator := int64(9 * (2*k + 3))
		term.Mul(term, new(big.Float).SetPrec(prec).SetInt64(numerator))
		term.Quo(term, new(big.Float).SetPrec(prec).SetInt64(denominator))
	}
	value := new(big.Float).SetPrec(prec).Mul(new(big.Float).SetPrec(prec).SetInt64(2), sum)
	ln2Cache[prec] = value
	return new(big.Float).Copy(value)
}
