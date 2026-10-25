package wgpu

import (
	"fmt"
	"math"
)

func ceBits(v float32) uint32 { return math.Float32bits(v) }

// Softmax cross-entropy in PrecisionFloat32Rounded, one thread per row. Every
// line is one correctly rounded operation or an exact two-term sum rounded
// once, in the order nn's softmax_ce_rounded.go defines, so both return the
// same bits. A row with a non-finite logit gives NaN.

const sceCommon = `
const ONE: u32 = 0x3f800000u;
const NEG_ONE: u32 = 0xbf800000u;

fn sce_sum2(a: u32, b: u32, c: u32, d: u32) -> u32 {
    var r: ExactRegister;
    exact_clear(&r);
    exact_add_product(&r, a, b);
    exact_add_product(&r, c, d);
    return exact_round(r);
}

fn sce_is_finite(bits: u32) -> bool {
    return ((bits >> 23u) & 0xffu) != 255u;
}
`

// The four steps run as separate kernels so the exponentials, which dominate,
// spread over one thread per logit instead of one per row.

// sceMaxKernel: params = rows, classes, 0, stride. mx[row] is the row maximum,
// or a NaN pattern when any logit of the row is not finite.
var sceMaxKernel = &kernel{name: "rounded-ce-max", nRead: 1, nWrite: 1, body: sceCommon + `
struct Params { rows: u32, classes: u32, unused: u32, stride: u32, }
@group(0) @binding(0) var<storage, read> x: array<u32>;
@group(0) @binding(1) var<storage, read_write> mx: array<u32>;
@group(0) @binding(2) var<uniform> p: Params;

@compute @workgroup_size(64)
fn main(@builtin(global_invocation_id) gid: vec3<u32>) {
    let row = gid.x + gid.y * p.stride;
    if (row >= p.rows) {
        return;
    }
    let base = row * p.classes;
    var finite = true;
    var m = bitcast<f32>(x[base]);
    for (var c: u32 = 0u; c < p.classes; c = c + 1u) {
        let bits = x[base + c];
        if (!sce_is_finite(bits)) {
            finite = false;
        }
        let v = bitcast<f32>(bits);
        if (v > m) {
            m = v;
        }
    }
    if (finite) {
        mx[row] = bitcast<u32>(m);
    } else {
        mx[row] = 0x7fc00000u;
    }
}
`}

// sceExpKernel: params = length (rows*classes), classes, 0, stride. One thread
// per logit: d = x - max and e = exp(d), or NaN for a row with a non-finite
// logit.
var sceExpKernel = &kernel{name: "rounded-ce-exp", nRead: 3, nWrite: 2, fixed: true, body: sceCommon + `
struct Params { length: u32, classes: u32, unused: u32, stride: u32, }
@group(0) @binding(0) var<storage, read> x: array<u32>;
@group(0) @binding(1) var<storage, read> mx: array<u32>;
@group(0) @binding(2) var<storage, read> consts: array<u32>;
@group(0) @binding(3) var<storage, read_write> dd: array<u32>;
@group(0) @binding(4) var<storage, read_write> ee: array<u32>;
@group(0) @binding(5) var<uniform> p: Params;

@compute @workgroup_size(64)
fn main(@builtin(global_invocation_id) gid: vec3<u32>) {
    let i = gid.x + gid.y * p.stride;
    if (i >= p.length) {
        return;
    }
    let m = mx[i / p.classes];
    if (!sce_is_finite(m)) {
        dd[i] = 0x7fc00000u;
        ee[i] = 0x7fc00000u;
        return;
    }
    let d = sce_sum2(x[i], ONE, NEG_ONE, m);
    dd[i] = d;
    ee[i] = exp_rounded(d);
}
`}

// sceRowKernel: params = rows, classes, 0, stride. s[row] is the exact sum of
// e, rounded once; loss[row] is log1p(o) - d[label], o the exact sum of the
// exponentials but the first peak's, so 1 + o is never rounded.
var sceRowKernel = &kernel{name: "rounded-ce-rows", nRead: 5, nWrite: 2, fixed: true, body: sceCommon + `
struct Params { rows: u32, classes: u32, unused: u32, stride: u32, }
@group(0) @binding(0) var<storage, read> x: array<u32>;
@group(0) @binding(1) var<storage, read> dd: array<u32>;
@group(0) @binding(2) var<storage, read> ee: array<u32>;
@group(0) @binding(3) var<storage, read> labels: array<u32>;
@group(0) @binding(4) var<storage, read> consts: array<u32>;
@group(0) @binding(5) var<storage, read_write> sums: array<u32>;
@group(0) @binding(6) var<storage, read_write> loss: array<u32>;
@group(0) @binding(7) var<uniform> p: Params;

// Orders float32 bit patterns as the floats compare, with -0 equal to +0.
fn sce_key(bits: u32) -> u32 {
    if ((bits & 0x7fffffffu) == 0u) {
        return 0x80000000u;
    }
    if ((bits >> 31u) == 1u) {
        return ~bits;
    }
    return bits | 0x80000000u;
}

@compute @workgroup_size(64)
fn main(@builtin(global_invocation_id) gid: vec3<u32>) {
    let row = gid.x + gid.y * p.stride;
    if (row >= p.rows) {
        return;
    }
    let base = row * p.classes;
    var best: u32 = 0u;
    var best_key = sce_key(x[base]);
    for (var c: u32 = 1u; c < p.classes; c = c + 1u) {
        let k = sce_key(x[base + c]);
        if (k > best_key) {
            best_key = k;
            best = c;
        }
    }
    var total: ExactRegister;
    var others: ExactRegister;
    exact_clear(&total);
    exact_clear(&others);
    for (var c: u32 = 0u; c < p.classes; c = c + 1u) {
        exact_add_product(&total, ee[base + c], ONE);
        if (c != best) {
            exact_add_product(&others, ee[base + c], ONE);
        }
    }
    sums[row] = exact_round(total);
    let l = log1p_rounded(exact_round(others));
    let li = labels[row];
    let dl = dd[base + li];
    loss[row] = sce_sum2(l, ONE, NEG_ONE, dl);
}
`}

// sceMeanKernel: params = rows, rows as float32 bits. One thread sums the row
// losses exactly, rounds once and divides by the row count.
var sceMeanKernel = &kernel{name: "rounded-ce-mean", nRead: 1, nWrite: 1, body: sceCommon + `
struct Params { rows: u32, rows_bits: u32, unused0: u32, unused1: u32, }
@group(0) @binding(0) var<storage, read> losses: array<u32>;
@group(0) @binding(1) var<storage, read_write> out: array<u32>;
@group(0) @binding(2) var<uniform> p: Params;

@compute @workgroup_size(1)
fn main() {
    var r: ExactRegister;
    exact_clear(&r);
    for (var i: u32 = 0u; i < p.rows; i = i + 1u) {
        exact_add_product(&r, losses[i], ONE);
    }
    out[0] = f32_div(exact_round(r), p.rows_bits);
}
`}

// sceGradKernel: params = length, classes, rows as float32 bits, stride. One
// thread per logit; the thread of a row's label also sums the row's other
// exponentials.
var sceGradKernel = &kernel{name: "rounded-ce-grad", nRead: 4, nWrite: 1, body: sceCommon + `
struct Params { length: u32, classes: u32, rows_bits: u32, stride: u32, }
@group(0) @binding(0) var<storage, read> ee: array<u32>;
@group(0) @binding(1) var<storage, read> sums: array<u32>;
@group(0) @binding(2) var<storage, read> labels: array<u32>;
@group(0) @binding(3) var<storage, read> upstream: array<u32>;
@group(0) @binding(4) var<storage, read_write> grad: array<u32>;
@group(0) @binding(5) var<uniform> p: Params;

@compute @workgroup_size(64)
fn main(@builtin(global_invocation_id) gid: vec3<u32>) {
    let i = gid.x + gid.y * p.stride;
    if (i >= p.length) {
        return;
    }
    let row = i / p.classes;
    let c = i % p.classes;
    let scale = f32_div(upstream[0], p.rows_bits);
    var pr: ExactRegister;
    exact_clear(&pr);
    if (c == labels[row]) {
        // p - 1 is -(sum of the others)/s. Written as p - 1 it is exactly 0
        // once the margin passes the float32 precision cliff.
        var others: ExactRegister;
        exact_clear(&others);
        let base = row * p.classes;
        for (var k: u32 = 0u; k < p.classes; k = k + 1u) {
            if (k != c) {
                exact_add_product(&others, ee[base + k], ONE);
            }
        }
        let q = f32_div(exact_round(others), sums[row]);
        exact_add_product(&pr, q ^ 0x80000000u, scale);
    } else {
        exact_add_product(&pr, f32_div(ee[i], sums[row]), scale);
    }
    grad[i] = exact_round(pr);
}
`}

// CEState holds the intermediates of a forward pass that the gradient reuses,
// so the exponentials are computed once.
type CEState struct {
	E, S *Buffer
}

// Release frees the state.
func (s *CEState) Release() {
	if s == nil {
		return
	}
	s.E.Release()
	s.S.Release()
}

// SoftmaxCrossEntropyRounded returns the mean loss, a one-value buffer, of
// logits [rows,classes] against int32 labels held as bit patterns, with the
// state the gradient needs.
func SoftmaxCrossEntropyRounded(logits, labels *Buffer, rows, classes int) (*Buffer, *CEState, error) {
	if rows <= 0 || classes <= 0 || logits.Len() != rows*classes || labels.Len() != rows {
		return nil, nil, fmt.Errorf("wgpu: cross-entropy buffers hold %d logits and %d labels for [%d,%d]", logits.Len(), labels.Len(), rows, classes)
	}
	consts, err := fixedConstants()
	if err != nil {
		return nil, nil, err
	}
	var temp []*Buffer
	release := func() {
		for _, b := range temp {
			b.Release()
		}
	}
	alloc := func(n int) (*Buffer, error) {
		b, err := NewBuffer(n)
		if err == nil {
			temp = append(temp, b)
		}
		return b, err
	}
	maxes, err := alloc(rows)
	if err != nil {
		return nil, nil, err
	}
	d, err := alloc(rows * classes)
	if err != nil {
		release()
		return nil, nil, err
	}
	e, err := alloc(rows * classes)
	if err != nil {
		release()
		return nil, nil, err
	}
	sums, err := alloc(rows)
	if err != nil {
		release()
		return nil, nil, err
	}
	losses, err := alloc(rows)
	if err != nil {
		release()
		return nil, nil, err
	}
	rg, ry, rstride := elementwiseGroups(rows)
	eg, ey, estride := elementwiseGroups(rows * classes)
	if err := sceMaxKernel.run([]*Buffer{logits}, []*Buffer{maxes}, [16]uint32{uint32(rows), uint32(classes), 0, rstride}, rg, ry); err != nil {
		release()
		return nil, nil, err
	}
	if err := sceExpKernel.run([]*Buffer{logits, maxes, consts}, []*Buffer{d, e}, [16]uint32{uint32(rows * classes), uint32(classes), 0, estride}, eg, ey); err != nil {
		release()
		return nil, nil, err
	}
	if err := sceRowKernel.run([]*Buffer{logits, d, e, labels, consts}, []*Buffer{sums, losses}, [16]uint32{uint32(rows), uint32(classes), 0, rstride}, rg, ry); err != nil {
		release()
		return nil, nil, err
	}
	out, err := NewBuffer(1)
	if err != nil {
		release()
		return nil, nil, err
	}
	if err := sceMeanKernel.run([]*Buffer{losses}, []*Buffer{out}, [16]uint32{uint32(rows), ceBits(float32(rows)), 0, 0}, 1, 1); err != nil {
		out.Release()
		release()
		return nil, nil, err
	}
	// maxes, d and losses are scratch; e and sums outlive the call.
	maxes.Release()
	d.Release()
	losses.Release()
	return out, &CEState{E: e, S: sums}, nil
}

// SoftmaxCrossEntropyGradRounded returns the gradient with respect to the
// logits, shaped like them, from the state of the forward pass.
func SoftmaxCrossEntropyGradRounded(state *CEState, labels, upstream *Buffer, rows, classes int) (*Buffer, error) {
	if rows <= 0 || classes <= 0 || state == nil || state.E.Len() != rows*classes || labels.Len() != rows || upstream.Len() != 1 {
		return nil, fmt.Errorf("wgpu: cross-entropy gradient buffers do not match [%d,%d]", rows, classes)
	}
	out, err := NewBuffer(rows * classes)
	if err != nil {
		return nil, err
	}
	gx, gy, stride := elementwiseGroups(rows * classes)
	if err := sceGradKernel.run([]*Buffer{state.E, state.S, labels, upstream}, []*Buffer{out},
		[16]uint32{uint32(rows * classes), uint32(classes), ceBits(float32(rows)), stride}, gx, gy); err != nil {
		out.Release()
		return nil, err
	}
	return out, nil
}
