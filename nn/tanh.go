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

// tanhSettleSteps is how far, in float64 steps, the float64 tanh must lie from
// every float32 midpoint for its float32 rounding to be certain.
const tanhSettleSteps = 16

// tanhHardCases holds the correctly rounded result for every |x| whose float64
// tanh lies within tanhSettleSteps of a float32 midpoint, keyed and valued by
// float32 bits. TestTanhIsCorrectlyRoundedExhaustive lists exactly these inputs.
var tanhHardCases = map[uint32]uint32{}

// Cody-Waite reduction constants for tanhFloat64. tanhLn2Hi keeps only the
// leading 31 significand bits of ln(2), so its low 21 mantissa bits are zero
// and every product with the k tanhFloat64 forms, which never exceeds 27, is
// exact; tanhLn2Lo carries the rest.
const (
	tanhLn2Hi  = 0x1.62e42fee00000p-1
	tanhLn2Lo  = 0x1.a39ef35793c76p-33
	tanhInvLn2 = 0x1.71547652b82fep0
)

// tanhC2 through tanhC15 are 1/k! for k = 2..15, each the nearest float64, the
// coefficients tanhExpm1Small sums.
const (
	tanhC2  = 0x1.0000000000000p-1
	tanhC3  = 0x1.5555555555555p-3
	tanhC4  = 0x1.5555555555555p-5
	tanhC5  = 0x1.1111111111111p-7
	tanhC6  = 0x1.6c16c16c16c17p-10
	tanhC7  = 0x1.a01a01a01a01ap-13
	tanhC8  = 0x1.a01a01a01a01ap-16
	tanhC9  = 0x1.71de3a556c734p-19
	tanhC10 = 0x1.27e4fb7789f5cp-22
	tanhC11 = 0x1.ae64567f544e4p-26
	tanhC12 = 0x1.1eed8eff8d898p-29
	tanhC13 = 0x1.6124613a86d09p-33
	tanhC14 = 0x1.93974a8c07c9dp-37
	tanhC15 = 0x1.ae7f3e733b81fp-41
)

// tanhFloat32 returns the correctly rounded value of tanh for one float32
// input, using a certified float64 fast path and an independent fallback.
func tanhFloat32(x float32) float32 {
	if x != x {
		return x
	}
	// 0x39000000 is 2^-13, 0x41180000 is 9.5, and the comparison is on the
	// bit patterns so it also catches +-Inf in the second range.
	bits := math.Float32bits(x)
	a := bits & 0x7fffffff
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

	t := tanhFloat64(float64(math.Float32frombits(a)))
	var y uint32
	if tanhFloat64Settles(t) {
		y = math.Float32bits(float32(t))
	} else if hard, ok := tanhHardCases[a]; ok {
		y = hard
	} else {
		// The exhaustive comparison shows tanhHardCases covers every input the
		// fast path cannot decide, so this arm is never reached. It stays so
		// that changing tanhFloat64 without refilling the table still returns
		// the correctly rounded answer.
		y = math.Float32bits(tanhHighPrecision(math.Float32frombits(a)))
	}
	return math.Float32frombits(y | bits&0x80000000)
}

// tanhMidpointSteps reports how far t is from the nearest float32 midpoint, in
// float64 steps, signed the same way as the mantissa: positive means t sits
// above the midpoint below it. For 2^-14 < |t| < 1, where every float32 is
// normal, a float32 step is 2^29 float64 steps, so the midpoints in t's binade
// are exactly the float64 values whose low 29 significand bits equal 1<<28, and
// the only midpoint outside it that could be near, the one just below the
// binade's lowest float32, is at least 2^27 float64 steps from any t in the
// binade. low - 1<<28 therefore counts the float64 steps from t to the nearest
// midpoint, and Ziv's method reduces to comparing that count against
// tanhSettleSteps.
func tanhMidpointSteps(t float64) int64 {
	return int64(math.Float64bits(t)&(1<<29-1)) - 1<<28
}

// tanhFloat64Settles reports whether float32(t) is the correctly rounded tanh
// when t is tanhFloat64's result for an input with 2^-13 <= |x| < 9.5. The
// margin it needs is tanhSettleSteps, and tanhFloat64's largest error over
// every float32 input is measured by TestTanhIsCorrectlyRoundedExhaustive.
// That measurement is a bound on every platform at once, because tanhFloat64
// uses only IEEE 754 addition, subtraction, multiplication, division and
// explicit conversions, which the Go specification requires to be rounded
// individually, so the same input gives the same bits everywhere.
func tanhFloat64Settles(t float64) bool {
	d := tanhMidpointSteps(t)
	return d > tanhSettleSteps || d < -tanhSettleSteps
}

// tanhExpm1Small returns expm1(u) for |u| up to about 0.35, which is the range
// Cody-Waite reduction leaves tanhFloat64 in. The Taylor series through 1/15!
// is evaluated in Estrin order, and every product carries an explicit float64
// conversion, which the Go specification rounds on its own, so no two of them
// can be fused.
func tanhExpm1Small(u float64) float64 {
	u2 := float64(u * u)
	u4 := float64(u2 * u2)
	u8 := float64(u4 * u4)
	p01 := tanhC2 + float64(u*tanhC3)
	p23 := tanhC4 + float64(u*tanhC5)
	p45 := tanhC6 + float64(u*tanhC7)
	p67 := tanhC8 + float64(u*tanhC9)
	p89 := tanhC10 + float64(u*tanhC11)
	p1011 := tanhC12 + float64(u*tanhC13)
	p1213 := tanhC14 + float64(u*tanhC15)
	q0 := p01 + float64(u2*p23)
	q1 := p45 + float64(u2*p67)
	q2 := p89 + float64(u2*p1011)
	r0 := q0 + float64(u4*q1)
	r1 := q2 + float64(u4*p1213)
	q := r0 + float64(u8*r1)
	return u + float64(u2*q)
}

// tanhFloat64 returns a float64 approximation of tanh for a float32 input a in
// [2^-13, 9.5), which is the range tanhFloat32 hands it. It writes the
// identity tanh(a) = -expm1(-2a) / (2 + expm1(-2a)), so only one small
// exponential is needed: with z = 2a, k = round(z/ln(2)) and r = z - k*ln(2),
// which is Cody-Waite reduction, expm1(-2a) = 2^-k*expm1(-r) + (2^-k - 1).
// The low bits of tanhLn2Hi are zero, so k*tanhLn2Hi is exact and r keeps the
// full precision of z; 2^-k is a power of two, so both terms of that identity
// are exact too and only the addition rounds, leaving tanhExpm1Small with the
// only approximation in the expression.
//
// Every product below carries an explicit float64 conversion, which the Go
// specification requires to be rounded to float64 and therefore forbids
// fusing with its neighbour, and the function otherwise uses only IEEE 754
// addition, subtraction, multiplication and division. So the same input gives
// the same float64 bits on every platform, unlike math.Tanh, whose arm64,
// amd64 and s390x implementations each contract multiply-add differently. An
// exhaustive comparison against the exact value on one machine therefore bounds
// the error on all of them.
func tanhFloat64(a float64) float64 {
	z := 2 * a
	k := int(float64(z*tanhInvLn2) + 0.5)
	hi := z - float64(float64(k)*tanhLn2Hi)
	r := hi - float64(float64(k)*tanhLn2Lo)
	scale := math.Float64frombits(uint64(1023-k) << 52)
	em1 := float64(scale*tanhExpm1Small(-r)) + (scale - 1)
	return -em1 / (2 + em1)
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
