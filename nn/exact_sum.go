package nn

import (
	"math"
	"math/bits"
)

// The accumulator holds an exact sum of float32 products as an integer
// scaled by 2^-exactSumScale. A product of two finite float32 values is
// M·2^k with M < 2^48 and -298 <= k <= 208, so it occupies bits [0, 554)
// of that integer. The integer is kept as exactSumDigits signed 32-bit
// digits stored in int64, so carries can be deferred; 640 bits leave room
// for the carries of any sum the accumulator is asked to hold.
const (
	exactSumScale          = 298
	exactSumDigits         = 20
	exactSumNormalizeEvery = 1 << 28
)

// exactAccumulator is a fixed-point register holding an exact sum of float32
// products. Adding a product is exact and order-free, so a caller may split
// the edges of one total across cores, threads or devices and read the same
// bits back. The sum is rounded once, when float32 reads it.
type exactAccumulator struct {
	digits  [exactSumDigits]int64
	pending int // additions since the last normalize
	nan     bool
	posInf  bool
	negInf  bool
}

// reset returns the accumulator to the empty total, so one accumulator can
// be reused for the next output without reallocating.
func (a *exactAccumulator) reset() {
	for i := range a.digits {
		a.digits[i] = 0
	}
	a.pending = 0
	a.nan = false
	a.posInf = false
	a.negInf = false
}

// addProduct adds the exact value of x*y. For finite operands, the product is
// M·2^k with M < 2^48 and -298 <= k <= 208, and it always lands inside the
// register. Carries are deferred until exactSumNormalizeEvery
// additions have piled up, which keeps the addition itself to three digit
// updates.
func (a *exactAccumulator) addProduct(x, y float32) {
	if x != x || y != y {
		a.nan = true
		return
	}

	xInf := math.IsInf(float64(x), 0)
	yInf := math.IsInf(float64(y), 0)
	if xInf || yInf {
		if x == 0 || y == 0 {
			a.nan = true
			return
		}
		if math.Signbit(float64(x)) != math.Signbit(float64(y)) {
			a.negInf = true
		} else {
			a.posInf = true
		}
		return
	}

	nx, mx, ex := decodeFiniteFloat32(x)
	ny, my, ey := decodeFiniteFloat32(y)
	if mx == 0 || my == 0 {
		return // the product is exactly zero, whatever the signs say
	}

	p := mx * my
	// The register counts in units of 2^-exactSumScale, so the product
	// contributes its significand at bit ex+ey+exactSumScale. That position
	// is in [0, 506], which is why the product spans at most three digits and
	// why digits[0] and digits[1] cannot overflow on the way in.
	shift := ex + ey + exactSumScale
	k := shift / 32
	off := uint(shift % 32)
	lo := (p & 0xffffffff) << off
	hi := (p >> 32) << off

	d0 := int64(lo & 0xffffffff)
	d1 := int64(lo>>32) + int64(hi&0xffffffff)
	d2 := int64(hi >> 32)
	if nx != ny {
		d0, d1, d2 = -d0, -d1, -d2
	}

	a.digits[k] += d0
	a.digits[k+1] += d1
	a.digits[k+2] += d2

	a.pending++
	if a.pending >= exactSumNormalizeEvery {
		a.normalize()
		a.pending = 0
	}
}

// normalize carries every digit into the next one, leaving digits[0] through
// digits[exactSumDigits-2] in [0, 2^32) and digits[exactSumDigits-1] holding
// the sign. The shift is arithmetic and so floors: a digit that went negative
// carries a negative amount and still leaves a remainder in [0, 2^32), which
// is what keeps the two's complement representation of a negative total exact.
func (a *exactAccumulator) normalize() {
	for i := 0; i < exactSumDigits-1; i++ {
		c := a.digits[i] >> 32
		a.digits[i] -= c << 32
		a.digits[i+1] += c
	}
}

// decodeFiniteFloat32 splits a finite float32 into its sign, an integer
// significand and an exponent, so that the value is
// (-1)^negative * mant * 2^exp. Subnormals and zero keep the significand they
// were stored with, at the smallest exponent, which is what makes a product
// that involves one exact too.
func decodeFiniteFloat32(x float32) (negative bool, mant uint64, exp int) {
	bits := math.Float32bits(x)
	e := (bits >> 23) & 0xff
	m := bits & 0x7fffff
	if e == 0 {
		return bits>>31 == 1, uint64(m), 1 - 150
	}
	return bits>>31 == 1, uint64(m) | 1<<23, int(e) - 150
}

// float32 rounds the total to the nearest float32, ties to even, and leaves
// the accumulator holding the exact value: it works on a copy, so a caller
// may read a total and keep adding to the same accumulator. A total that is
// exactly zero, including an empty one, is +0.
func (a *exactAccumulator) float32() float32 {
	if a.nan || (a.posInf && a.negInf) {
		return math.Float32frombits(0x7fc00000)
	}
	if a.posInf {
		return math.Float32frombits(0x7f800000)
	}
	if a.negInf {
		return math.Float32frombits(0xff800000)
	}

	work := *a
	work.normalize()

	negative := work.digits[exactSumDigits-1] < 0
	if negative {
		for i := range work.digits {
			work.digits[i] = -work.digits[i]
		}
		work.normalize() // the two's complement negation, carried again
	}

	msb := -1
	for i := exactSumDigits - 1; i >= 0; i-- {
		if work.digits[i] != 0 {
			msb = 32*i + bits.Len64(uint64(work.digits[i])) - 1
			break
		}
	}
	if msb < 0 {
		return math.Float32frombits(0)
	}

	// Keep the 24 bits at the top of the total, and let everything below
	// them decide the rounding alone. The smallest normal float32 is 2^-126,
	// whose least significant bit sits at bit 149 of the register, so a
	// subnormal total pins the window there instead of sliding down with it
	// and rounding a value that is not on the subnormal grid.
	l := msb - 23
	if l < 149 {
		l = 149
	}
	kept := exactSumWindow(&work.digits, l, msb-l+1)
	roundBit := exactSumBit(&work.digits, l-1) == 1
	sticky := exactSumSticky(&work.digits, l-1)
	if roundBit && (sticky || kept&1 == 1) {
		kept++ // ties to even: only round up on an odd kept or a dirty tail
	}
	if kept == 1<<24 {
		kept >>= 1
		l++
	}

	var out uint32
	if kept < 1<<23 {
		// Below the smallest normal float32 the significand is the value in
		// units of 2^-149 and the exponent field is zero.
		out = uint32(kept)
	} else if biased := l - 148; biased >= 0xff {
		out = 0x7f800000 // a total this large rounds to infinity
	} else {
		out = uint32(biased)<<23 | uint32(kept&0x7fffff)
	}
	if negative {
		out |= 1 << 31
	}
	return math.Float32frombits(out)
}

// exactSumBit reports bit k of a normalized total, counting bit 0 as the
// least significant. Bits outside the register read as zero.
func exactSumBit(d *[exactSumDigits]int64, k int) int {
	if k < 0 || k >= exactSumDigits*32 {
		return 0
	}
	return int((uint64(d[k/32]) >> uint(k%32)) & 1)
}

// exactSumWindow returns the n bits of a normalized total starting at bit lo
// as one integer, with the bit at lo in the result's bit 0. float32 calls it
// with nonnegative digits, lo >= 0, and 1 <= n <= 24, so the window spans at
// most two 32-bit digits. A window starting outside the register reads as zero.
func exactSumWindow(d *[exactSumDigits]int64, lo, n int) uint64 {
	i := lo / 32
	if i >= exactSumDigits {
		return 0
	}
	off := uint(lo % 32)
	w := uint64(d[i])
	if i+1 < exactSumDigits {
		w |= uint64(d[i+1]) << 32
	}
	return (w >> off) & (1<<uint(n) - 1)
}

// exactSumSticky reports whether any bit below lo is set, which is the
// "round up only if something was dropped" half of ties to even. float32 calls
// it with nonnegative digits and lo >= 0. It checks complete 32-bit digits
// first, then the remaining low bits of the digit containing lo.
func exactSumSticky(d *[exactSumDigits]int64, lo int) bool {
	j := lo / 32
	for k := 0; k < j && k < exactSumDigits; k++ {
		if d[k] != 0 {
			return true
		}
	}
	if j < exactSumDigits && d[j]&((1<<uint(lo%32))-1) != 0 {
		return true
	}
	return false
}

// float64ProductSum is the fast path for exact sums of float32 products. It
// reaches the same answer as the exact accumulator, in floating point, and
// falls back to it whenever it cannot prove its own.
//
// The product of two float32 values is exact in float64: 24 + 24 significand
// bits fit in 53, and the product's exponent range (2^-298 to 2^256) sits
// well inside float64's, so widening the operands multiplies without a
// rounding. A float64 total therefore differs from the exact total only by the
// rounding of its own additions, and recursive summation of n terms carries a
// forward error of at most gamma(n-1)*sum|p| with gamma(k) = k*u/(1-k*u) and
// u = 2^-53 (Higham, Accuracy and Stability of Numerical Algorithms, 2nd ed.,
// §4.2). abs accumulates sum|p| the same way, so both halves of that bound
// are available in floating point.
//
// result reads the float32 f the float64 total rounds to and asks where the
// exact total can be. If the whole interval [S-B, S+B] the bound allows lies
// strictly between the two midpoints that would round to a neighbouring
// float32 instead, then every value in the interval rounds to f, so f is the
// correctly rounded exact total. An interval holding no midpoint has one
// answer; that is what makes the cheap path provable rather than merely close.
// This is Ziv's method: a fast path behind a provable criterion, with the
// exact accumulator behind it for the totals the criterion cannot settle.
type float64ProductSum struct {
	sum, abs float64
	n        int
}

// reset returns the fast path to the empty total, so one can be reused for
// the next output without reallocating.
func (f *float64ProductSum) reset() {
	f.sum = 0
	f.abs = 0
	f.n = 0
}

// add folds one product into the running totals. The widening makes x*y exact,
// so only the two additions round, and both are the ones §4.2 bounds.
func (f *float64ProductSum) add(x, y float32) {
	p := float64(x) * float64(y)
	f.sum += p
	f.abs += math.Abs(p)
	f.n++
}

// result returns the correctly rounded exact total of everything added, and
// whether the fast path could prove it. A false second result means the
// caller must reach for the exact accumulator: this is a performance
// decision, never a difference in the answer. Non-finite totals refuse, as do
// the totals whose sign of zero or overflow boundary the exact accumulator
// owns.
func (f *float64ProductSum) result() (float32, bool) {
	if f.n == 0 {
		return float32(0), true // an empty total is positive zero
	}
	if math.IsNaN(f.sum) || math.IsInf(f.sum, 0) || math.IsNaN(f.abs) || math.IsInf(f.abs, 0) {
		return 0, false
	}
	if f.n == 1 {
		// One product rounds nothing: the total is the exact value, and Go's
		// float64 to float32 conversion rounds to nearest with ties to even
		// across the whole range, subnormals and overflow included.
		if f.sum == 0 {
			return float32(0), true
		}
		return float32(f.sum), true
	}

	// The first term is four times gamma(n-1)*sum|p| with room to spare, so
	// it covers that bound and the rounding abs itself carries; the second
	// covers the rounding of the two comparisons below. Both hold for every
	// n below 2^40, far more terms than any edge list holds.
	bound := f.abs*float64(f.n)*0x1p-51 + math.Abs(f.sum)*0x1p-52

	out := float32(f.sum)
	if out == 0 || math.IsInf(float64(out), 0) || math.Abs(float64(out)) == math.MaxFloat32 {
		// An exact zero's sign, and the edge where a total rounds to
		// infinity, belong to the exact accumulator.
		return 0, false
	}

	// The midpoints to the neighbours of out. Each is the average of two
	// adjacent float32 values, whose sum needs 25 significand bits and is
	// therefore exact in float64.
	prev := math.Nextafter32(out, float32(math.Inf(-1)))
	next := math.Nextafter32(out, float32(math.Inf(1)))
	lo := (float64(out) + float64(prev)) / 2
	hi := (float64(out) + float64(next)) / 2
	if f.sum-bound > lo && f.sum+bound < hi {
		return out, true
	}
	return 0, false
}
