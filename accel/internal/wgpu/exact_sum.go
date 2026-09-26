package wgpu

// exactSumWGSL is a WGSL library, with no entry point, for the exact sum of
// float32 products that nn's CPU accumulator computes (nn/exact_sum.go). A
// register counts in units of 2^-298, like the CPU's, and holds 27 digits of
// 24 bits in u32 with carries deferred for up to 64 additions. Float32 values
// cross it as bit patterns and every operation is unsigned, so the result does
// not depend on how a shader compiler treats floating point or signed
// integers. Kernels prepend it to their own source. It uses no select():
// gogpu/naga v0.19 writes a scalar select as an unparenthesized ternary in
// Metal, which changes its meaning inside a larger expression.
const exactSumWGSL = `
const EXACT_MASK: u32 = 0xffffffu;
const EXACT_FLAG_NAN: u32 = 1u;
const EXACT_FLAG_POS_INF: u32 = 2u;
const EXACT_FLAG_NEG_INF: u32 = 4u;

struct ExactRegister { digits: array<u32, 27>, pending: u32, flags: u32, }

// Clears every digit, counter, and flag so the register represents exact zero
// with no carry history.
fn exact_clear(r: ptr<function, ExactRegister>) {
    for (var i: u32 = 0u; i < 27u; i = i + 1u) {
        (*r).digits[i] = 0u;
    }
    (*r).pending = 0u;
    (*r).flags = 0u;
}

// Propagates every digit into the next one. The low digits then hold magnitudes
// below 2^24, the top digit carries the sign, and explicit sign extension makes
// subtraction of a whole negative limb exact. The carry is the signed value
// arithmetic-shifted right by 24, with the high 8 bits sign-extended through
// 0u - (v >> 31u), which is all ones when v is negative. It avoids select()
// because gogpu/naga v0.19 writes a select used as an operand as an
// unparenthesized ternary, which would re-associate the expression.
fn exact_normalize(r: ptr<function, ExactRegister>) {
    for (var i: u32 = 0u; i < 26u; i = i + 1u) {
        let v = (*r).digits[i];
        let carry = (v >> 24u) | ((0u - (v >> 31u)) << 8u);
        (*r).digits[i] = v & EXACT_MASK;
        (*r).digits[i + 1u] = (*r).digits[i + 1u] + carry;
    }
}

// Adds the exact product of two binary32 bit patterns. The 48-bit significand
// product spans at most three digits. At most 64 finite nonzero products are
// added between passes, so every digit's true magnitude stays below
// 65 * 2^24 < 2^31 and its two's-complement encoding cannot wrap.
fn exact_add_product(r: ptr<function, ExactRegister>, x: u32, y: u32) {
    let ex = (x >> 23u) & 0xffu;
    let fx = x & 0x7fffffu;
    let ey = (y >> 23u) & 0xffu;
    let fy = y & 0x7fffffu;

    if ((ex == 255u && fx != 0u) || (ey == 255u && fy != 0u)) {
        (*r).flags = (*r).flags | EXACT_FLAG_NAN;
        return;
    }

    if (ex == 255u || ey == 255u) {
        if ((x & 0x7fffffffu) == 0u || (y & 0x7fffffffu) == 0u) {
            (*r).flags = (*r).flags | EXACT_FLAG_NAN;
        } else if (((x ^ y) >> 31u) == 1u) {
            (*r).flags = (*r).flags | EXACT_FLAG_NEG_INF;
        } else {
            (*r).flags = (*r).flags | EXACT_FLAG_POS_INF;
        }
        return;
    }

    var mx = fx;
    if (ex != 0u) {
        mx = fx | 0x800000u;
    }
    var my = fy;
    if (ey != 0u) {
        my = fy | 0x800000u;
    }
    if (mx == 0u || my == 0u) {
        return;
    }

    let shift = max(ex, 1u) + max(ey, 1u) - 2u;
    let k = shift / 24u;
    let off = shift % 24u;

    let a0 = mx & 0xfffu;
    let a1 = mx >> 12u;
    let b0 = my & 0xfffu;
    let b1 = my >> 12u;
    let low = a0 * b0;
    let mid = a0 * b1 + a1 * b0;
    let high = a1 * b1;
    let t = low + ((mid & 0xfffu) << 12u);
    let plo = t & EXACT_MASK;
    let phi = high + (mid >> 12u) + (t >> 24u);

    var d0 = (plo << off) & EXACT_MASK;
    var d1 = ((plo >> (24u - off)) | (phi << off)) & EXACT_MASK;
    var d2 = phi >> (24u - off);
    if (((x ^ y) >> 31u) == 1u) {
        d0 = 0u - d0;
        d1 = 0u - d1;
        d2 = 0u - d2;
    }

    (*r).digits[k] = (*r).digits[k] + d0;
    (*r).digits[k + 1u] = (*r).digits[k + 1u] + d1;
    (*r).digits[k + 2u] = (*r).digits[k + 2u] + d2;
    (*r).pending = (*r).pending + 1u;
    if ((*r).pending >= 64u) {
        exact_normalize(r);
        (*r).pending = 0u;
    }
}

// Normalizes both sides, adds their exact totals and special flags, then
// normalizes the destination again. Both sides are normalized first so every
// digit being added is below 2^24 and the sum stays far inside 2^31. Splitting a
// total this way cannot change its value or its eventual rounding.
fn exact_merge(r: ptr<function, ExactRegister>, other: ExactRegister) {
    exact_normalize(r);
    var o = other;
    exact_normalize(&o);
    for (var i: u32 = 0u; i < 27u; i = i + 1u) {
        (*r).digits[i] = (*r).digits[i] + o.digits[i];
    }
    (*r).flags = (*r).flags | o.flags;
    exact_normalize(r);
    (*r).pending = 0u;
}

// Rounds the exact total once to nearest-even. It works on a normalized copy,
// obtains a magnitude through unsigned subtraction when the total is negative,
// and combines the round bit with every discarded lower bit before returning
// the output bit pattern.
fn exact_round(r: ExactRegister) -> u32 {
    if ((r.flags & EXACT_FLAG_NAN) != 0u ||
        ((r.flags & EXACT_FLAG_POS_INF) != 0u && (r.flags & EXACT_FLAG_NEG_INF) != 0u)) {
        return 0x7fc00000u;
    }
    if ((r.flags & EXACT_FLAG_POS_INF) != 0u) {
        return 0x7f800000u;
    }
    if ((r.flags & EXACT_FLAG_NEG_INF) != 0u) {
        return 0xff800000u;
    }

    var w = r;
    exact_normalize(&w);
    let negative = (w.digits[26] & 0x80000000u) != 0u;
    if (negative) {
        for (var i: u32 = 0u; i < 27u; i = i + 1u) {
            w.digits[i] = 0u - w.digits[i];
        }
        exact_normalize(&w);
    }

    var msb: u32 = 0u;
    var found: bool = false;
    for (var j: u32 = 0u; j < 27u; j = j + 1u) {
        let i = 26u - j;
        if (w.digits[i] != 0u) {
            msb = 24u * i + firstLeadingBit(w.digits[i]);
            found = true;
            break;
        }
    }
    if (!found) {
        return 0u;
    }

    var l: u32 = 149u;
    if (msb >= 172u) {
        l = msb - 23u;
    }
    var kept: u32 = 0u;
    if (msb >= l) {
        let i = l / 24u;
        let o = l % 24u;
        var hi: u32 = 0u;
        if (i + 1u < 27u) {
            hi = w.digits[i + 1u];
        }
        let n = msb - l + 1u;
        kept = ((w.digits[i] >> o) | (hi << (24u - o))) & ((1u << n) - 1u);
    }

    let roundBit = ((w.digits[(l - 1u) / 24u] >> ((l - 1u) % 24u)) & 1u) != 0u;
    let q = (l - 1u) / 24u;
    var sticky: bool = false;
    for (var i: u32 = 0u; i < q; i = i + 1u) {
        if (w.digits[i] != 0u) {
            sticky = true;
        }
    }
    if ((w.digits[q] & ((1u << ((l - 1u) % 24u)) - 1u)) != 0u) {
        sticky = true;
    }

    if (roundBit && (sticky || (kept & 1u) == 1u)) {
        kept = kept + 1u;
    }
    if (kept == 0x1000000u) {
        kept = kept >> 1u;
        l = l + 1u;
    }

    var out: u32 = 0u;
    if (kept < 0x800000u) {
        out = kept;
    } else {
        let biased = l - 148u;
        if (biased >= 255u) {
            out = 0x7f800000u;
        } else {
            out = (biased << 23u) | (kept & 0x7fffffu);
        }
    }
    if (negative) {
        out = out | 0x80000000u;
    }
    return out;
}
`
