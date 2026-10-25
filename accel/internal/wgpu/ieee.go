package wgpu

// ieeeWGSL is a WGSL library, with no entry point, for float32 division and
// square root as exact integer algorithms on bit patterns. Both return the
// IEEE 754 correctly rounded result (round to nearest, ties to even, subnormals
// kept, signs of zero and infinity as IEEE gives them) without using a floating
// point operation, so no shader compiler can change them. A NaN result is the
// canonical quiet NaN.
const ieeeWGSL = `
// Rounds a 24-bit significand with a guard bit and a sticky bit and packs it
// with a biased exponent that is at least 1. A carry out of the significand
// moves to the next exponent, and an exponent of 255 or more is infinity.
fn ieee_pack_normal(sign: u32, biased: i32, kept_in: u32, guard: bool, sticky: bool) -> u32 {
    var kept = kept_in;
    var be = biased;
    if (guard && (sticky || (kept & 1u) == 1u)) {
        kept = kept + 1u;
    }
    if (kept == 0x1000000u) {
        kept = 0x800000u;
        be = be + 1;
    }
    if (be >= 255) {
        return sign | 0x7f800000u;
    }
    return sign | (u32(be) << 23u) | (kept & 0x7fffffu);
}

// Packs a value whose biased exponent is below 1: a subnormal or zero. ext is
// the 24-bit significand followed by the guard bit and a sticky bit.
fn ieee_pack_subnormal(sign: u32, biased: i32, kept: u32, guard: bool, sticky: bool) -> u32 {
    let sh = u32(1 - biased);
    if (sh >= 25u) {
        return sign;
    }
    var gbit = 0u;
    if (guard) {
        gbit = 2u;
    }
    var sbit = 0u;
    if (sticky) {
        sbit = 1u;
    }
    let ext = (kept << 2u) | gbit | sbit;
    var q = ext >> (sh + 2u);
    let round_bit = ((ext >> (sh + 1u)) & 1u) == 1u;
    let low = (ext & ((1u << (sh + 1u)) - 1u)) != 0u;
    if (round_bit && (low || (q & 1u) == 1u)) {
        q = q + 1u;
    }
    return sign | q;
}

// Splits a finite nonzero float32 into a significand in [2^23, 2^24) and an
// unbiased exponent e with value = m * 2^(e - 23).
fn ieee_decode(x: u32) -> vec2<i32> {
    let ex = (x >> 23u) & 0xffu;
    let fx = x & 0x7fffffu;
    if (ex == 0u) {
        // countLeadingZeros is not used: naga compiles it to FindUMsb without the
        // 31 - x, so on Vulkan it returns the bit index instead of the count.
        let shift = 23u - firstLeadingBit(fx);
        return vec2<i32>(i32(fx << shift), 1 - i32(shift) - 127);
    }
    return vec2<i32>(i32(fx | 0x800000u), i32(ex) - 127);
}

fn f32_div(x: u32, y: u32) -> u32 {
    let sign = (x ^ y) & 0x80000000u;
    let ex = (x >> 23u) & 0xffu;
    let ey = (y >> 23u) & 0xffu;
    let nan_x = ex == 255u && (x & 0x7fffffu) != 0u;
    let nan_y = ey == 255u && (y & 0x7fffffu) != 0u;
    if (nan_x || nan_y) {
        return 0x7fc00000u;
    }
    let xinf = ex == 255u;
    let yinf = ey == 255u;
    let xzero = (x & 0x7fffffffu) == 0u;
    let yzero = (y & 0x7fffffffu) == 0u;
    if ((xinf && yinf) || (xzero && yzero)) {
        return 0x7fc00000u;
    }
    if (xinf) {
        return sign | 0x7f800000u;
    }
    if (yinf || xzero) {
        return sign;
    }
    if (yzero) {
        return sign | 0x7f800000u;
    }
    let dx = ieee_decode(x);
    let dy = ieee_decode(y);
    let my = u32(dy.x);
    var r = u32(dx.x);
    var q: u32 = 0u;
    for (var i: u32 = 0u; i < 27u; i = i + 1u) {
        q = q << 1u;
        if (r >= my) {
            r = r - my;
            q = q | 1u;
        }
        r = r << 1u;
    }
    let rem = r != 0u;
    var e = dx.y - dy.y;
    var kept: u32;
    var guard: bool;
    var sticky: bool;
    if (q >= 0x4000000u) {
        kept = q >> 3u;
        guard = ((q >> 2u) & 1u) == 1u;
        sticky = (q & 3u) != 0u || rem;
    } else {
        kept = q >> 2u;
        guard = ((q >> 1u) & 1u) == 1u;
        sticky = (q & 1u) != 0u || rem;
        e = e - 1;
    }
    let biased = e + 127;
    if (biased >= 1) {
        return ieee_pack_normal(sign, biased, kept, guard, sticky);
    }
    return ieee_pack_subnormal(sign, biased, kept, guard, sticky);
}

fn f32_sqrt(x: u32) -> u32 {
    let ex = (x >> 23u) & 0xffu;
    if (ex == 255u && (x & 0x7fffffu) != 0u) {
        return 0x7fc00000u;
    }
    if ((x & 0x7fffffffu) == 0u) {
        return x;
    }
    if ((x >> 31u) == 1u) {
        return 0x7fc00000u;
    }
    if (ex == 255u) {
        return 0x7f800000u;
    }
    let d = ieee_decode(x);
    var m = u32(d.x);
    var k = d.y - 23;
    if ((k & 1) != 0) {
        m = m << 1u;
        k = k - 1;
    }
    var rem: u32 = 0u;
    var q: u32 = 0u;
    for (var j: u32 = 0u; j < 27u; j = j + 1u) {
        let pair_index = 26u - j;
        var pair: u32 = 0u;
        if (2u * pair_index >= 28u) {
            pair = (m >> (2u * pair_index - 28u)) & 3u;
        }
        rem = (rem << 2u) | pair;
        let t = (q << 2u) | 1u;
        if (rem >= t) {
            rem = rem - t;
            q = (q << 1u) | 1u;
        } else {
            q = q << 1u;
        }
    }
    let inexact = rem != 0u;
    var e = k / 2;
    var kept: u32;
    var guard: bool;
    var sticky: bool;
    if (q >= 0x4000000u) {
        kept = q >> 3u;
        guard = ((q >> 2u) & 1u) == 1u;
        sticky = (q & 3u) != 0u || inexact;
        e = e + 12;
    } else {
        kept = q >> 2u;
        guard = ((q >> 1u) & 1u) == 1u;
        sticky = (q & 1u) != 0u || inexact;
        e = e + 11;
    }
    return ieee_pack_normal(0u, e + 127, kept, guard, sticky);
}
`
