package wgpu

import (
	"fmt"
	"math"
	"math/big"
	"strings"
	"sync"
)

// fixedWGSL is a WGSL library, with no entry point, for 128-bit unsigned
// fixed-point arithmetic with 112 fractional bits, held as eight 16-bit
// digits in u32. Digit 7 is the integer part, so a value ranges over
// [0, 65536) with a resolution of 2^-112. Every operation is on unsigned
// integers, so no shader compiler can change a result. It carries the
// constants the exp and log kernels need, computed once in Go with math/big.
//
// A fixed multiplication discards the columns below the result's lowest digit,
// which loses at most eight units of 2^-112 per product; the exp and log
// kernels run at most thirty products, so the accumulated error stays below
// 2^-100, far under the 2^-60 that correct rounding of a float32 needs (the
// exhaustive tests are the proof).
const fixedFrac = 112

var (
	fixedOnce  sync.Once
	fixedSrc   string
	fixedWords []uint32

	fixedBufferMu sync.Mutex
	fixedBuffer   *Buffer
)

const (
	expTerms = 31 // coefficients 1/n! for n = 0..30
	atTerms  = 23 // coefficients 1/(2n+1) for n = 0..22
)

// fixedConstants returns the device buffer holding the library's constants,
// uploading it on first use. It lives for the process.
func fixedConstants() (*Buffer, error) {
	fixedLibrary()
	fixedBufferMu.Lock()
	defer fixedBufferMu.Unlock()
	if fixedBuffer != nil && !fixedBuffer.Released() {
		return fixedBuffer, nil
	}
	values := make([]float32, len(fixedWords))
	for i, w := range fixedWords {
		values[i] = math.Float32frombits(w)
	}
	buffer, err := Upload(values)
	if err != nil {
		return nil, err
	}
	fixedBuffer = buffer
	return buffer, nil
}

func fixedLibrary() string {
	fixedOnce.Do(func() {
		const prec = 600
		scale := new(big.Float).SetPrec(prec).SetInt(new(big.Int).Lsh(big.NewInt(1), fixedFrac))
		digits := func(v *big.Float) [8]uint32 {
			scaled := new(big.Float).SetPrec(prec).Mul(v, scale)
			scaled.Add(scaled, big.NewFloat(0.5))
			integer, _ := scaled.Int(nil)
			var out [8]uint32
			mask := big.NewInt(0xffff)
			for i := 0; i < 8; i++ {
				out[i] = uint32(new(big.Int).And(new(big.Int).Rsh(integer, uint(16*i)), mask).Uint64())
			}
			return out
		}
		format := func(list [][8]uint32) string {
			var parts []string
			for _, d := range list {
				for _, x := range d {
					parts = append(parts, fmt.Sprintf("%du", x))
				}
			}
			return strings.Join(parts, ", ")
		}
		// ln 2 = 2 atanh(1/3) = sum 2 / ((2k+1) 3^(2k+1))
		ln2 := new(big.Float).SetPrec(prec)
		power := new(big.Float).SetPrec(prec).Quo(big.NewFloat(1), big.NewFloat(3))
		ninth := new(big.Float).SetPrec(prec).Quo(big.NewFloat(1), big.NewFloat(9))
		for k := 0; k < 300; k++ {
			term := new(big.Float).SetPrec(prec).Quo(power, new(big.Float).SetPrec(prec).SetInt64(int64(2*k+1)))
			ln2.Add(ln2, term)
			power.Mul(power, ninth)
		}
		ln2.Mul(ln2, big.NewFloat(2))
		var expC, atC [][8]uint32
		fact := new(big.Float).SetPrec(prec).SetInt64(1)
		for n := 0; n < expTerms; n++ {
			if n > 0 {
				fact.Mul(fact, new(big.Float).SetPrec(prec).SetInt64(int64(n)))
			}
			expC = append(expC, digits(new(big.Float).SetPrec(prec).Quo(big.NewFloat(1), fact)))
		}
		for n := 0; n < atTerms; n++ {
			atC = append(atC, digits(new(big.Float).SetPrec(prec).Quo(big.NewFloat(1), new(big.Float).SetPrec(prec).SetInt64(int64(2*n+1)))))
		}
		_ = format
		// Layout of the constants buffer: ln 2, then the exp coefficients,
		// then the atanh coefficients, eight digits each.
		for _, d := range append(append([][8]uint32{digits(ln2)}, expC...), atC...) {
			fixedWords = append(fixedWords, d[:]...)
		}
		fixedSrc = fixedBody + fixedFunctions
	})
	return fixedSrc
}

const fixedBody = `
struct Fx { d: array<u32, 8>, }

fn fx_zero() -> Fx {
    var r: Fx;
    for (var i: u32 = 0u; i < 8u; i = i + 1u) {
        r.d[i] = 0u;
    }
    return r;
}

fn fx_add(a: Fx, b: Fx) -> Fx {
    var r: Fx;
    var carry: u32 = 0u;
    for (var i: u32 = 0u; i < 8u; i = i + 1u) {
        let x = a.d[i];
        let y = b.d[i];
        let t = x + y + carry;
        r.d[i] = t & 0xffffu;
        carry = t >> 16u;
    }
    return r;
}

// a - b, wrapping modulo 2^128.
fn fx_sub(a: Fx, b: Fx) -> Fx {
    var r: Fx;
    var borrow: u32 = 0u;
    for (var i: u32 = 0u; i < 8u; i = i + 1u) {
        let x = a.d[i];
        let y = b.d[i];
        let t = x + 0x10000u - y - borrow;
        r.d[i] = t & 0xffffu;
        borrow = 1u - (t >> 16u);
    }
    return r;
}

// True when a >= b, comparing as unsigned 128-bit values.
fn fx_ge(a: Fx, b: Fx) -> bool {
    for (var j: u32 = 0u; j < 8u; j = j + 1u) {
        let i = 7u - j;
        let x = a.d[i];
        let y = b.d[i];
        if (x > y) {
            return true;
        }
        if (x < y) {
            return false;
        }
    }
    return true;
}

// The value as a signed two's-complement number is negative.
fn fx_negative(a: Fx) -> bool {
    return (a.d[7] & 0x8000u) != 0u;
}

fn fx_is_zero(a: Fx) -> bool {
    var any: u32 = 0u;
    for (var i: u32 = 0u; i < 8u; i = i + 1u) {
        let x = a.d[i];
        any = any | x;
    }
    return any == 0u;
}

// (a * b) >> 112, dropping the columns below the result.
fn fx_mul(a: Fx, b: Fx) -> Fx {
    var lo: array<u32, 16>;
    var hi: array<u32, 16>;
    for (var k: u32 = 0u; k < 16u; k = k + 1u) {
        lo[k] = 0u;
        hi[k] = 0u;
    }
    for (var i: u32 = 0u; i < 8u; i = i + 1u) {
        for (var j: u32 = 0u; j < 8u; j = j + 1u) {
            if (i + j >= 7u) {
                let x = a.d[i];
                let y = b.d[j];
                let p = x * y;
                let lw = lo[i + j];
                lo[i + j] = lw + (p & 0xffffu);
                let hw = hi[i + j];
                hi[i + j] = hw + (p >> 16u);
            }
        }
    }
    var r: Fx;
    var carry: u32 = 0u;
    for (var k: u32 = 7u; k < 15u; k = k + 1u) {
        var below: u32 = 0u;
        if (k > 7u) {
            below = hi[k - 1u];
        }
        let own = lo[k];
        let t = own + below + carry;
        r.d[k - 7u] = t & 0xffffu;
        carry = t >> 16u;
    }
    return r;
}

// a * s for a small integer s < 65536 (the result must stay below 2^16).
fn fx_mul_small(a: Fx, s: u32) -> Fx {
    var r: Fx;
    var carry: u32 = 0u;
    for (var i: u32 = 0u; i < 8u; i = i + 1u) {
        let x = a.d[i];
        let t = x * s + carry;
        r.d[i] = t & 0xffffu;
        carry = t >> 16u;
    }
    return r;
}

// v << s for a 24-bit v and s < 100, as a fixed-point value.
fn fx_from_shifted(v: u32, s: u32) -> Fx {
    var r = fx_zero();
    let q = s / 16u;
    let b = s % 16u;
    // v << b has at most 39 bits: split over three digits.
    let low = (v << b) & 0xffffu;
    let mid = (v >> (16u - b)) & 0xffffu;
    var top: u32 = 0u;
    if (b > 0u) {
        top = (v >> (32u - b)) & 0xffffu;
    }
    r.d[q] = low;
    if (q + 1u < 8u) {
        r.d[q + 1u] = mid;
    }
    if (q + 2u < 8u) {
        r.d[q + 2u] = top;
    }
    return r;
}

fn fx_bit(a: Fx, p: u32) -> u32 {
    let x = a.d[p >> 4u];
    return (x >> (p & 15u)) & 1u;
}

// Index of the highest set bit, or 0 for zero.
fn fx_msb(a: Fx) -> u32 {
    for (var j: u32 = 0u; j < 8u; j = j + 1u) {
        let i = 7u - j;
        let x = a.d[i];
        if (x != 0u) {
            return 16u * i + firstLeadingBit(x);
        }
    }
    return 0u;
}

// True when any bit below position p (exclusive) is set.
fn fx_any_below(a: Fx, p: u32) -> bool {
    let q = p >> 4u;
    for (var i: u32 = 0u; i < q; i = i + 1u) {
        let x = a.d[i];
        if (x != 0u) {
            return true;
        }
    }
    let m = (1u << (p & 15u)) - 1u;
    let top = a.d[q];
    return (top & m) != 0u;
}

// The 24 bits that end at bit p, as an integer.
fn fx_kept24(a: Fx, p: u32) -> u32 {
    var v: u32 = 0u;
    for (var i: u32 = 0u; i < 24u; i = i + 1u) {
        v = (v << 1u) | fx_bit(a, p - i);
    }
    return v;
}

// The constants live in a read-only storage buffer the kernel binds as
// consts: ln 2 at word 0, 31 exp coefficients from word 8, 23 atanh
// coefficients from word 256.
fn fx_const(idx: u32, which: u32) -> Fx {
    var r: Fx;
    var base: u32 = 8u + idx * 8u;
    if (which == 1u) {
        base = 256u + idx * 8u;
    }
    for (var i: u32 = 0u; i < 8u; i = i + 1u) {
        r.d[i] = consts[base + i];
    }
    return r;
}

fn fx_ln2() -> Fx {
    var r: Fx;
    for (var i: u32 = 0u; i < 8u; i = i + 1u) {
        r.d[i] = consts[i];
    }
    return r;
}
`

const fixedFunctions = `
// Packs the fixed-point value p (at least 2^112, below 2^113) times 2^k as a
// float32 bit pattern: 24 bits, a guard bit and a sticky bit, then the shared
// packing for normal and subnormal results.
fn fx_pack_exp(p: Fx, k: i32) -> u32 {
    let kept = fx_kept24(p, 112u);
    let guard = fx_bit(p, 88u) == 1u;
    let sticky = fx_any_below(p, 88u);
    let biased = k + 127;
    if (biased >= 1) {
        return ieee_pack_normal(0u, biased, kept, guard, sticky);
    }
    return ieee_pack_subnormal(0u, biased, kept, guard, sticky);
}

fn exp_rounded(x: u32) -> u32 {
    let ax = x & 0x7fffffffu;
    let negative = (x >> 31u) == 1u;
    if (ax > 0x7f800000u) {
        return 0x7fc00000u;
    }
    if (ax == 0x7f800000u) {
        if (negative) {
            return 0u;
        }
        return 0x7f800000u;
    }
    if (ax < 0x32000000u) {
        return 0x3f800000u;
    }
    if (!negative && ax >= 0x42b20000u) {
        return 0x7f800000u;
    }
    if (negative && ax >= 0x42d00000u) {
        return 0u;
    }
    let ex = (ax >> 23u) & 0xffu;
    let m = (ax & 0x7fffffu) | 0x800000u;
    let big_x = fx_from_shifted(m, ex - 38u);
    let ln2 = fx_ln2();
    // k0 estimates |x| / ln 2, rounded up for negative x and down for positive.
    let estimate = bitcast<f32>(ax) * 1.4426950408889634;
    var k0: u32 = u32(estimate);
    if (negative) {
        k0 = k0 + 1u;
    }
    var r: Fx;
    if (negative) {
        r = fx_sub(fx_mul_small(ln2, k0), big_x);
        if (fx_negative(r)) {
            k0 = k0 + 1u;
            r = fx_add(r, ln2);
        }
        if (fx_ge(r, ln2)) {
            k0 = k0 - 1u;
            r = fx_sub(r, ln2);
        }
    } else {
        r = fx_sub(big_x, fx_mul_small(ln2, k0));
        if (fx_negative(r)) {
            k0 = k0 - 1u;
            r = fx_add(r, ln2);
        }
        if (fx_ge(r, ln2)) {
            k0 = k0 + 1u;
            r = fx_sub(r, ln2);
        }
    }
    var acc = fx_const(30u, 0u);
    for (var j: u32 = 0u; j < 30u; j = j + 1u) {
        let n = 29u - j;
        acc = fx_add(fx_const(n, 0u), fx_mul(r, acc));
    }
    var k = i32(k0);
    if (negative) {
        k = -k;
    }
    return fx_pack_exp(acc, k);
}

fn log_rounded(x: u32) -> u32 {
    let ex = (x >> 23u) & 0xffu;
    if (ex == 255u && (x & 0x7fffffu) != 0u) {
        return 0x7fc00000u;
    }
    if ((x & 0x7fffffffu) == 0u) {
        return 0xff800000u;
    }
    if ((x >> 31u) == 1u) {
        return 0x7fc00000u;
    }
    if (ex == 255u) {
        return 0x7f800000u;
    }
    let d = ieee_decode(x);
    let mant = u32(d.x);
    var e = d.y;
    var num: u32;
    var den: u32;
    var z_negative = false;
    if (mant < 11863283u) {
        num = mant - 0x800000u;
        den = mant + 0x800000u;
    } else {
        num = 0x1000000u - mant;
        den = mant + 0x1000000u;
        z_negative = true;
        e = e + 1;
    }
    // z = num / den to 112 fractional bits by restoring division.
    var z = fx_zero();
    var rem = num;
    for (var j: u32 = 1u; j <= 112u; j = j + 1u) {
        rem = rem << 1u;
        if (rem >= den) {
            rem = rem - den;
            let bit = 112u - j;
            z.d[bit >> 4u] = z.d[bit >> 4u] | (1u << (bit & 15u));
        }
    }
    let w = fx_mul(z, z);
    var q = fx_const(22u, 1u);
    for (var j: u32 = 0u; j < 22u; j = j + 1u) {
        let n = 21u - j;
        q = fx_add(fx_const(n, 1u), fx_mul(w, q));
    }
    var s = fx_mul(z, q);
    s = fx_add(s, s);
    var abs_e = e;
    var e_negative = false;
    if (e < 0) {
        abs_e = -e;
        e_negative = true;
    }
    let a = fx_mul_small(fx_ln2(), u32(abs_e));
    var mag: Fx;
    var negative_result = false;
    if (e_negative == z_negative) {
        mag = fx_add(a, s);
        negative_result = e_negative;
    } else if (fx_ge(a, s)) {
        mag = fx_sub(a, s);
        negative_result = e_negative;
    } else {
        mag = fx_sub(s, a);
        negative_result = z_negative;
    }
    if (fx_is_zero(mag)) {
        return 0u;
    }
    let p = fx_msb(mag);
    let kept = fx_kept24(mag, p);
    let guard = fx_bit(mag, p - 24u) == 1u;
    let sticky = fx_any_below(mag, p - 24u);
    var sign = 0u;
    if (negative_result) {
        sign = 0x80000000u;
    }
    return ieee_pack_normal(sign, i32(p) - 112 + 127, kept, guard, sticky);
}
`
