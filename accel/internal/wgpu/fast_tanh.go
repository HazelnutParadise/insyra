package wgpu

import (
	"math/big"
	"math/bits"
)

// The fast tanh works in 64-bit unsigned fixed point with 58 fractional bits,
// each value held as two u32 words. For 2^-13 <= |x| < 9.5 it writes
// z = 2|x| = n ln2/64 + r with 0 <= r < ln2/64, so e^-z = 2^-(n>>6) *
// 2^(-(n&63)/64) * e^-r, takes the middle factor from a 64-entry table and
// e^-r from nine terms of its series, and divides 1 - e^-z by 1 + e^-z
// through two Newton steps on the reciprocal. Every value it rounds is within
// three units of 2^-58 of the exact one: measured on 200,000 inputs against a
// 200-bit oracle, with the reciprocal's start moved by 2^-18 either way, which
// covers how far a device's f32 division can be from the start the host
// takes. fastTanhBound is 64 units, so an input it settles is correctly
// rounded; an input within that of a rounding midpoint returns 0 and takes the
// 128-bit path. On the host model of this arithmetic 243 of the 135,790,592
// positive inputs in range do.
const (
	fastTanhConstBase = 440 // after ln 2, the exp and the atanh coefficients
	fastTanhFrac      = 58
)

// fastTanhConstants returns ln2/64 at scale 2^70 (ln2 at scale 2^64), then
// 2^(-j/64) at scale 2^58 for j = 0..63, each as its low then high word.
func fastTanhConstants(ln2 *big.Float, prec uint) []uint32 {
	round := func(v *big.Float, scale int) uint64 {
		s := new(big.Float).SetPrec(prec).SetMantExp(v, scale)
		s.Add(s, big.NewFloat(0.5))
		i, _ := s.Int(nil)
		return i.Uint64()
	}
	words := func(v uint64) []uint32 { return []uint32{uint32(v), uint32(v >> 32)} }
	out := words(round(ln2, 64))
	for j := 0; j < 64; j++ {
		x := new(big.Float).SetPrec(prec).Mul(ln2, big.NewFloat(float64(j)))
		x.Quo(x, big.NewFloat(64))
		// e^-x by its series, which converges fast for x < ln 2.
		sum := new(big.Float).SetPrec(prec).SetInt64(1)
		term := new(big.Float).SetPrec(prec).SetInt64(1)
		for n := int64(1); n < 300; n++ {
			term.Mul(term, x)
			term.Quo(term, new(big.Float).SetPrec(prec).SetInt64(-n))
			sum.Add(sum, term)
		}
		out = append(out, words(round(sum, fastTanhFrac))...)
	}
	return out
}

// fastTanhModel is the host model of tanh_fast's arithmetic, with the
// reciprocal started from y0 (scale 2^58). It returns tanh|x| at scale 2^58.
// The tests use it to measure the error bound and count the inputs that fall
// back; the device result is checked against nn.Tanh, not against this.
func fastTanhModel(ax uint32, constants []uint32, y0 uint64) uint64 {
	word := func(i int) uint64 {
		return uint64(constants[fastTanhConstBase+2*i]) | uint64(constants[fastTanhConstBase+2*i+1])<<32
	}
	mul := func(a, b uint64) uint64 {
		hi, lo := bits.Mul64(a, b)
		return hi<<(64-fastTanhFrac) | lo>>fastTanhFrac
	}
	const one = uint64(1) << fastTanhFrac
	m := uint64(ax&0x7fffff | 0x800000)
	z := m << ((ax >> 23) - 91)
	n, rem := bits.Div64(z>>52, z<<12, word(0))
	r := rem >> 12
	p := one
	for k := uint64(9); k >= 1; k-- {
		p = one - mul(r, p)/k
	}
	e := mul(word(1+int(n&63)), p) >> (n >> 6)
	num, den := one-e, one+e
	y := y0
	for i := 0; i < 2; i++ {
		y = mul(y, 2*one-mul(den, y))
	}
	return mul(num, y)
}

// fastTanhSettle is tanh_fast's rounding: the float32 bits of q (scale
// 2^58), or false when q is within fastTanhBound of a rounding midpoint.
func fastTanhSettle(q uint64) (uint32, bool) {
	p := bits.Len64(q) - 1
	shift := uint(p - 23)
	low := q & (1<<shift - 1)
	half := uint64(1) << (shift - 1)
	d := low - half
	if low < half {
		d = half - low
	}
	if d <= fastTanhBound {
		return 0, false
	}
	kept := uint32(q >> shift)
	if low > half {
		kept++
	}
	biased := uint32(p + 69)
	if kept == 1<<24 {
		kept >>= 1
		biased++
	}
	return biased<<23 | kept&0x7fffff, true
}

const fastTanhBound = 64

const fastTanhWGSL = `
// 64-bit unsigned values as vec2<u32>(low, high); see fast_tanh.go.

fn u64_mul32(a: u32, b: u32) -> vec2<u32> {
    let a0 = a & 0xffffu;
    let a1 = a >> 16u;
    let b0 = b & 0xffffu;
    let b1 = b >> 16u;
    let p00 = a0 * b0;
    let p01 = a0 * b1;
    let p10 = a1 * b0;
    let p11 = a1 * b1;
    let mid = (p00 >> 16u) + (p01 & 0xffffu) + (p10 & 0xffffu);
    return vec2<u32>((p00 & 0xffffu) | (mid << 16u), p11 + (p01 >> 16u) + (p10 >> 16u) + (mid >> 16u));
}

fn u64_add(a: vec2<u32>, b: vec2<u32>) -> vec2<u32> {
    let lo = a.x + b.x;
    var carry = 0u;
    if (lo < a.x) {
        carry = 1u;
    }
    return vec2<u32>(lo, a.y + b.y + carry);
}

fn u64_sub(a: vec2<u32>, b: vec2<u32>) -> vec2<u32> {
    var borrow = 0u;
    if (a.x < b.x) {
        borrow = 1u;
    }
    return vec2<u32>(a.x - b.x, a.y - b.y - borrow);
}

fn u64_less(a: vec2<u32>, b: vec2<u32>) -> bool {
    if (a.y != b.y) {
        return a.y < b.y;
    }
    return a.x < b.x;
}

// a >> s for s < 64.
fn u64_shr(a: vec2<u32>, s: u32) -> vec2<u32> {
    if (s == 0u) {
        return a;
    }
    if (s >= 32u) {
        return vec2<u32>(a.y >> (s - 32u), 0u);
    }
    return vec2<u32>((a.x >> s) | (a.y << (32u - s)), a.y >> s);
}

// a << s for s < 64, dropping what leaves the top.
fn u64_shl(a: vec2<u32>, s: u32) -> vec2<u32> {
    if (s == 0u) {
        return a;
    }
    if (s >= 32u) {
        return vec2<u32>(0u, a.x << (s - 32u));
    }
    return vec2<u32>(a.x << s, (a.y << s) | (a.x >> (32u - s)));
}

// floor(a * b / 2^58), for a product below 2^122.
fn u64_mulq(a: vec2<u32>, b: vec2<u32>) -> vec2<u32> {
    let ll = u64_mul32(a.x, b.x);
    let lh = u64_mul32(a.x, b.y);
    let hl = u64_mul32(a.y, b.x);
    let hh = u64_mul32(a.y, b.y);
    // Words w1, w2, w3 of the 128-bit product; w0 = ll.x is below the result.
    var w1 = ll.y;
    var c1 = 0u;
    w1 = w1 + lh.x;
    if (w1 < lh.x) {
        c1 = c1 + 1u;
    }
    w1 = w1 + hl.x;
    if (w1 < hl.x) {
        c1 = c1 + 1u;
    }
    var w2 = lh.y;
    var c2 = 0u;
    w2 = w2 + hl.y;
    if (w2 < hl.y) {
        c2 = c2 + 1u;
    }
    w2 = w2 + hh.x;
    if (w2 < hh.x) {
        c2 = c2 + 1u;
    }
    w2 = w2 + c1;
    if (w2 < c1) {
        c2 = c2 + 1u;
    }
    let w3 = hh.y + c2;
    return vec2<u32>((w1 >> 26u) | (w2 << 6u), (w2 >> 26u) | (w3 << 6u));
}

// floor(a / k) for 1 <= k <= 9.
fn u64_div_small(a: vec2<u32>, k: u32) -> vec2<u32> {
    var rem = 0u;
    var cur = (rem << 16u) | (a.y >> 16u);
    let q3 = cur / k;
    rem = cur % k;
    cur = (rem << 16u) | (a.y & 0xffffu);
    let q2 = cur / k;
    rem = cur % k;
    cur = (rem << 16u) | (a.x >> 16u);
    let q1 = cur / k;
    rem = cur % k;
    cur = (rem << 16u) | (a.x & 0xffffu);
    let q0 = cur / k;
    return vec2<u32>((q1 << 16u) | q0, (q3 << 16u) | q2);
}

fn fast_word(i: u32) -> vec2<u32> {
    let base = 440u + 2u * i;
    return vec2<u32>(consts[base], consts[base + 1u]);
}

// tanh|x| as float32 bits for 2^-13 <= |x| < 9.5, or 0 when the result is
// within 64 units of 2^-58 of a rounding midpoint.
fn tanh_fast(ax: u32) -> u32 {
    let one = vec2<u32>(0u, 1u << 26u);
    let c = fast_word(0u);
    let m = (ax & 0x7fffffu) | 0x800000u;
    let z = u64_shl(vec2<u32>(m, 0u), (ax >> 23u) - 91u);
    // z * 2^12 as three words: z < 2^63, so the top word holds 11 bits.
    let z0 = z.x << 12u;
    let z1 = (z.y << 12u) | (z.x >> 20u);
    let z2 = z.y >> 20u;
    // n = floor(z / (ln2/64)), from an f32 estimate one below, then exact.
    var n = u32(bitcast<f32>(ax) * 184.66496523378731);
    if (n > 0u) {
        n = n - 1u;
    }
    let pl = u64_mul32(n, c.x);
    let ph = u64_mul32(n, c.y);
    var p1 = pl.y + ph.x;
    var p2 = ph.y;
    if (p1 < ph.x) {
        p2 = p2 + 1u;
    }
    // rem = z * 2^12 - n * c, which an estimate one below keeps non-negative.
    var r0 = z0 - pl.x;
    var b = 0u;
    if (z0 < pl.x) {
        b = 1u;
    }
    var r1 = z1 - p1 - b;
    var b2 = 0u;
    if (z1 < p1 || (z1 == p1 && b == 1u)) {
        b2 = 1u;
    }
    var r2 = z2 - p2 - b2;
    for (var i: u32 = 0u; i < 4u; i = i + 1u) {
        if (r2 == 0u && u64_less(vec2<u32>(r0, r1), c)) {
            break;
        }
        let low = u64_sub(vec2<u32>(r0, r1), c);
        if (u64_less(vec2<u32>(r0, r1), c)) {
            r2 = r2 - 1u;
        }
        r0 = low.x;
        r1 = low.y;
        n = n + 1u;
    }
    let r = u64_shr(vec2<u32>(r0, r1), 12u);
    var p = one;
    for (var j: u32 = 0u; j < 9u; j = j + 1u) {
        let k = 9u - j;
        p = u64_sub(one, u64_div_small(u64_mulq(r, p), k));
    }
    let e = u64_shr(u64_mulq(fast_word(1u + (n & 63u)), p), n >> 6u);
    let num = u64_sub(one, e);
    let den = u64_add(one, e);
    let df = f32(den.y) * (1.0 / 67108864.0);
    var y = vec2<u32>(0u, u32((1.0 / df) * 67108864.0));
    let two = vec2<u32>(0u, 1u << 27u);
    for (var it: u32 = 0u; it < 2u; it = it + 1u) {
        y = u64_mulq(y, u64_sub(two, u64_mulq(den, y)));
    }
    let q = u64_mulq(num, y);
    // q >= 2^46, so its top bit is in the high word.
    let top = 32u + firstLeadingBit(q.y);
    let shift = top - 23u;
    var low: vec2<u32>;
    var half: vec2<u32>;
    if (shift >= 32u) {
        low = vec2<u32>(q.x, q.y & ((1u << (shift - 32u)) - 1u));
    } else {
        low = vec2<u32>(q.x & ((1u << shift) - 1u), 0u);
    }
    if (shift - 1u >= 32u) {
        half = vec2<u32>(0u, 1u << (shift - 33u));
    } else {
        half = vec2<u32>(1u << (shift - 1u), 0u);
    }
    var d: vec2<u32>;
    let above = u64_less(half, low);
    if (above) {
        d = u64_sub(low, half);
    } else {
        d = u64_sub(half, low);
    }
    if (d.y == 0u && d.x <= 64u) {
        return 0u;
    }
    var kept = u64_shr(q, shift).x;
    if (above) {
        kept = kept + 1u;
    }
    var biased = top + 69u;
    if (kept == 0x1000000u) {
        kept = kept >> 1u;
        biased = biased + 1u;
    }
    return (biased << 23u) | (kept & 0x7fffffu);
}
`
