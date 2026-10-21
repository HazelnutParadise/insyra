package nn

import (
	"math"
	"math/big"
)

const (
	expLogSettleSteps = 32
	expLn2Hi          = 0x1.62e42fee00000p-1
	expLn2Lo          = 0x1.a39ef35793c76p-33
	expInvLn2         = 0x1.71547652b82fep0
	logSqrt2          = 0x1.6a09e667f3bcdp0
)

var expTaylorCoeff = [...]float64{
	1, 1, 1.0 / 2, 1.0 / 6, 1.0 / 24, 1.0 / 120, 1.0 / 720,
	1.0 / 5040, 1.0 / 40320, 1.0 / 362880, 1.0 / 3628800,
	1.0 / 39916800, 1.0 / 479001600, 1.0 / 6227020800,
	1.0 / 87178291200, 1.0 / 1307674368000, 1.0 / 20922789888000,
	1.0 / 355687428096000, 1.0 / 6402373705728000,
	1.0 / 121645100408832000, 1.0 / 2432902008176640000,
}

// expFloat32 returns the correctly rounded float32 value of e^x. The
// deterministic float64 path is used when its result is farther than
// expLogSettleSteps float64 steps from either adjacent float32 midpoint.
func expFloat32(x float32) float32 {
	if math.IsNaN(float64(x)) {
		return x
	}
	if math.IsInf(float64(x), 1) {
		return float32(math.Inf(1))
	}
	if math.IsInf(float64(x), -1) {
		return 0
	}
	if x < -104 {
		return 0
	}
	if x > 89 {
		return float32(math.Inf(1))
	}
	t := expFloat64(float64(x))
	if expLogFloat64Settles(t) {
		return float32(t)
	}
	if result, ok := expLogHardCases[expLogKey(0, x)]; ok {
		return math.Float32frombits(result)
	}
	return expHighPrecision(x)
}

// logFloat32 returns the correctly rounded float32 value of log(x).
func logFloat32(x float32) float32 {
	if math.IsNaN(float64(x)) {
		return x
	}
	if x == 0 {
		return float32(math.Inf(-1))
	}
	if x < 0 {
		return float32(math.NaN())
	}
	if math.IsInf(float64(x), 1) {
		return float32(math.Inf(1))
	}
	if x == 1 {
		return 0
	}
	t := logFloat64(float64(x))
	if expLogFloat64Settles(t) {
		return float32(t)
	}
	if result, ok := expLogHardCases[expLogKey(1, x)]; ok {
		return math.Float32frombits(result)
	}
	return logHighPrecision(x)
}

func expLogKey(which uint32, x float32) uint64 {
	return uint64(which)<<32 | uint64(math.Float32bits(x))
}

// expFloat64 reduces x = k*ln(2)+r, evaluates exp(r) by a degree-20 Taylor
// polynomial, and scales by an exact power of two. All arithmetic is IEEE
// + - * /; explicit conversions prevent multiply-add contraction.
func expFloat64(x float64) float64 {
	k := int(float64(x*expInvLn2) + 0.5)
	if x < 0 {
		k = int(float64(x*expInvLn2) - 0.5)
	}
	hi := x - float64(float64(k)*expLn2Hi)
	r := hi - float64(float64(k)*expLn2Lo)
	// Horner evaluation of sum(r^n/n!), n=0..20.
	p := expTaylorCoeff[20]
	for n := 19; n >= 0; n-- {
		p = float64(p*r) + expTaylorCoeff[n]
	}
	scale := math.Float64frombits(uint64(k+1023) << 52)
	return float64(p * scale)
}

// logFloat64 decomposes x into 2^e*m with m in [sqrt(2)/2,sqrt(2)), then
// evaluates 2*atanh((m-1)/(m+1)) and adds e*ln(2) in two pieces.
func logFloat64(x float64) float64 {
	bits := math.Float64bits(x)
	exponent := int((bits >> 52) & 0x7ff)
	frac := bits & ((uint64(1) << 52) - 1)
	e := exponent - 1023
	if exponent == 0 {
		x *= 0x1p54
		bits = math.Float64bits(x)
		exponent = int((bits >> 52) & 0x7ff)
		frac = bits & ((uint64(1) << 52) - 1)
		e = exponent - 1023 - 54
	}
	m := math.Float64frombits(uint64(1023)<<52 | frac)
	if m >= logSqrt2 {
		m *= 0.5
		e++
	}
	z := (m - 1) / (m + 1)
	tz2 := float64(z * z)
	term, sum := z, z
	for odd := 3; odd <= 49; odd += 2 {
		term = float64(term * tz2)
		sum += term / float64(odd)
	}
	logM := 2 * sum
	ef := float64(e)
	hi := float64(ef * expLn2Hi)
	lo := float64(ef * expLn2Lo)
	return (logM + hi) + lo
}

// expLogFloat64Settles checks distances to both adjacent float32 midpoints in
// float64 representable steps. This also handles the constant-spaced float32
// subnormal interval and its zero boundary.
func expLogFloat64Settles(t float64) bool {
	if math.IsNaN(t) || math.IsInf(t, 0) {
		return true
	}
	f := float32(t)
	fv := float64(f)
	var lo, hi float64
	if fv <= t {
		lo = fv
		hi = float64(math.Nextafter32(f, float32(math.Inf(1))))
	} else {
		lo = float64(math.Nextafter32(f, float32(math.Inf(-1))))
		hi = fv
	}
	min := uint64(^uint64(0))
	if !math.IsInf(lo, 0) && !math.IsInf(hi, 0) {
		mid := lo + (hi-lo)/2
		if d := orderedFloat64Distance(t, mid); d < min {
			min = d
		}
	}
	if !math.IsInf(hi, 0) {
		next := float64(math.Nextafter32(float32(hi), float32(math.Inf(1))))
		mid := hi + (next-hi)/2
		if d := orderedFloat64Distance(t, mid); d < min {
			min = d
		}
	}
	if !math.IsInf(lo, 0) {
		prev := float64(math.Nextafter32(float32(lo), float32(math.Inf(-1))))
		mid := prev + (lo-prev)/2
		if d := orderedFloat64Distance(t, mid); d < min {
			min = d
		}
	}
	// At the float32 overflow boundary the adjacent value is infinity; its
	// finite rounding midpoint is maxFloat32 + 2^103.
	if math.IsInf(hi, 1) {
		mid := 0x1.ffffffp127
		if d := orderedFloat64Distance(t, mid); d < min {
			min = d
		}
	}
	if math.IsInf(lo, -1) {
		mid := -0x1.ffffffp127
		if d := orderedFloat64Distance(t, mid); d < min {
			min = d
		}
	}
	return min > expLogSettleSteps
}

func orderedFloat64Bits(x float64) uint64 {
	b := math.Float64bits(x)
	if b>>63 != 0 {
		return ^b
	}
	return b | (uint64(1) << 63)
}

func orderedFloat64Distance(a, b float64) uint64 {
	x, y := orderedFloat64Bits(a), orderedFloat64Bits(b)
	if x < y {
		return y - x
	}
	return x - y
}

func expHighPrecision(x float32) float32 {
	result := expHighPrecisionExact(x)
	out, _ := result.Float32()
	return out
}

func expHighPrecisionExact(x float32) *big.Float {
	const prec = uint(384)
	if x > 100 {
		return new(big.Float).SetPrec(prec).SetInf(false)
	}
	if x < -110 {
		return new(big.Float).SetPrec(prec).SetInt64(0)
	}
	xb := new(big.Float).SetPrec(prec).SetFloat64(float64(x))
	ln2 := expLogLn2Big(prec)
	quot := new(big.Float).SetPrec(prec).Quo(xb, ln2)
	kf, _ := quot.Float64()
	k := int64(kf + 0.5)
	if kf < 0 {
		k = int64(kf - 0.5)
	}
	r := new(big.Float).SetPrec(prec).Sub(xb, new(big.Float).SetPrec(prec).Mul(new(big.Float).SetPrec(prec).SetInt64(k), ln2))
	one := new(big.Float).SetPrec(prec).SetInt64(1)
	sum, term := new(big.Float).SetPrec(prec).Set(one), new(big.Float).SetPrec(prec).Set(one)
	for n := int64(1); n < 1200; n++ {
		term.Mul(term, r)
		term.Quo(term, new(big.Float).SetPrec(prec).SetInt64(n))
		next := new(big.Float).SetPrec(prec).Add(sum, term)
		if next.Cmp(sum) == 0 {
			break
		}
		sum.Set(next)
	}
	sum.SetMantExp(sum, int(k))
	return sum
}

func logHighPrecision(x float32) float32 {
	result := logHighPrecisionExact(x)
	out, _ := result.Float32()
	return out
}

func logHighPrecisionExact(x float32) *big.Float {
	const prec = uint(384)
	p := uint(prec + 32)
	v := new(big.Float).SetPrec(p).SetFloat64(float64(x))
	m := new(big.Float).SetPrec(p)
	e := v.MantExp(m)
	m.Mul(m, new(big.Float).SetPrec(p).SetInt64(2))
	e--
	if m.Cmp(new(big.Float).SetPrec(p).SetFloat64(logSqrt2)) >= 0 {
		m.Quo(m, new(big.Float).SetPrec(p).SetInt64(2))
		e++
	}
	one := new(big.Float).SetPrec(p).SetInt64(1)
	z := new(big.Float).SetPrec(p).Quo(new(big.Float).SetPrec(p).Sub(m, one), new(big.Float).SetPrec(p).Add(m, one))
	z2 := new(big.Float).SetPrec(p).Mul(z, z)
	term := new(big.Float).SetPrec(p).Set(z)
	sum := new(big.Float).SetPrec(p).Set(z)
	for odd := int64(3); odd < 20000; odd += 2 {
		term.Mul(term, z2)
		add := new(big.Float).SetPrec(p).Quo(term, new(big.Float).SetPrec(p).SetInt64(odd))
		next := new(big.Float).SetPrec(p).Add(sum, add)
		if next.Cmp(sum) == 0 {
			break
		}
		sum.Set(next)
	}
	logM := new(big.Float).SetPrec(p).Mul(sum, new(big.Float).SetPrec(p).SetInt64(2))
	ln2 := expLogLn2Big(p)
	result := new(big.Float).SetPrec(p).Add(logM, new(big.Float).SetPrec(p).Mul(ln2, new(big.Float).SetPrec(p).SetInt64(int64(e))))
	return result
}

func expLogLn2Big(prec uint) *big.Float {
	// ln(2)=2*atanh(1/3), with its convergent power series.
	one := new(big.Float).SetPrec(prec).SetInt64(1)
	z := new(big.Float).SetPrec(prec).Quo(one, new(big.Float).SetPrec(prec).SetInt64(3))
	z2 := new(big.Float).SetPrec(prec).Mul(z, z)
	term, sum := new(big.Float).SetPrec(prec).Set(z), new(big.Float).SetPrec(prec).Set(z)
	for odd := int64(3); odd < 20000; odd += 2 {
		term.Mul(term, z2)
		add := new(big.Float).SetPrec(prec).Quo(term, new(big.Float).SetPrec(prec).SetInt64(odd))
		next := new(big.Float).SetPrec(prec).Add(sum, add)
		if next.Cmp(sum) == 0 {
			break
		}
		sum.Set(next)
	}
	return sum.Mul(sum, new(big.Float).SetPrec(prec).SetInt64(2))
}
