package wgpu

import (
	"fmt"
	"math"
)

// The kernels below compute in the exact-sum register (exact_sum.go): every
// output is the exact value of its operation rounded once to nearest-even
// float32, and an exactly-zero result is +0. They are the device half of
// nn's PrecisionFloat32Rounded; the CPU half is in package nn.

// matMulRoundedKernel: params = m, k, n, flags (bit 0 transA, bit 1 transB).
// Each workgroup computes a 16x16 tile of the output, staging 16x16 tiles of
// both operands in workgroup memory so each is read from device memory once
// per tile instead of once per output. Zero bit patterns pad the edges; a
// product with a zero factor adds nothing to the register.
var matMulRoundedKernel = &kernel{name: "rounded-matmul", nRead: 2, nWrite: 1, body: `
struct Params { m: u32, k: u32, n: u32, flags: u32, }
@group(0) @binding(0) var<storage, read> a: array<u32>;
@group(0) @binding(1) var<storage, read> b: array<u32>;
@group(0) @binding(2) var<storage, read_write> c: array<u32>;
@group(0) @binding(3) var<uniform> p: Params;

var<workgroup> tile_a: array<u32, 256>;
var<workgroup> tile_b: array<u32, 256>;

@compute @workgroup_size(16, 16)
fn main(@builtin(global_invocation_id) gid: vec3<u32>, @builtin(local_invocation_id) lid: vec3<u32>) {
    let col = gid.x;
    let row = gid.y;
    let transA = (p.flags & 1u) != 0u;
    let transB = (p.flags & 2u) != 0u;
    var r: ExactRegister;
    exact_clear(&r);
    let tiles = (p.k + 15u) / 16u;
    for (var t: u32 = 0u; t < tiles; t = t + 1u) {
        let ka = t * 16u + lid.x;
        var av: u32 = 0u;
        if (row < p.m && ka < p.k) {
            var ai = row * p.k + ka;
            if (transA) {
                ai = ka * p.m + row;
            }
            av = a[ai];
        }
        tile_a[lid.y * 16u + lid.x] = av;
        let kb = t * 16u + lid.y;
        var bv: u32 = 0u;
        if (col < p.n && kb < p.k) {
            var bi = kb * p.n + col;
            if (transB) {
                bi = col * p.k + kb;
            }
            bv = b[bi];
        }
        tile_b[lid.y * 16u + lid.x] = bv;
        workgroupBarrier();
        for (var kk: u32 = 0u; kk < 16u; kk = kk + 1u) {
            exact_add_product(&r, tile_a[lid.y * 16u + kk], tile_b[kk * 16u + lid.x]);
        }
        workgroupBarrier();
    }
    if (row < p.m && col < p.n) {
        c[row * p.n + col] = exact_round(r);
    }
}
`}

// addRoundedKernel: params = length, bLength, 0, stride. out[i] = a[i] + b[i % bLength].
var addRoundedKernel = &kernel{name: "rounded-add", nRead: 2, nWrite: 1, body: `
struct Params { length: u32, b_length: u32, unused: u32, stride: u32, }
@group(0) @binding(0) var<storage, read> a: array<u32>;
@group(0) @binding(1) var<storage, read> b: array<u32>;
@group(0) @binding(2) var<storage, read_write> c: array<u32>;
@group(0) @binding(3) var<uniform> p: Params;

@compute @workgroup_size(64)
fn main(@builtin(global_invocation_id) gid: vec3<u32>) {
    let i = gid.x + gid.y * p.stride;
    if (i >= p.length) {
        return;
    }
    var r: ExactRegister;
    exact_clear(&r);
    exact_add_product(&r, a[i], 0x3f800000u);
    exact_add_product(&r, b[i % p.b_length], 0x3f800000u);
    c[i] = exact_round(r);
}
`}

// sumRowsRoundedKernel: params = m, n. out[j] = sum over rows of x[i*n+j].
var sumRowsRoundedKernel = &kernel{name: "rounded-sum-rows", nRead: 1, nWrite: 1, body: `
struct Params { m: u32, n: u32, unused0: u32, unused1: u32, }
@group(0) @binding(0) var<storage, read> x: array<u32>;
@group(0) @binding(1) var<storage, read_write> c: array<u32>;
@group(0) @binding(2) var<uniform> p: Params;

@compute @workgroup_size(64)
fn main(@builtin(global_invocation_id) gid: vec3<u32>) {
    let col = gid.x;
    if (col >= p.n) {
        return;
    }
    var r: ExactRegister;
    exact_clear(&r);
    for (var row: u32 = 0u; row < p.m; row = row + 1u) {
        exact_add_product(&r, x[row * p.n + col], 0x3f800000u);
    }
    c[col] = exact_round(r);
}
`}

// sgdRoundedKernel: params = length, negated rate bits, 0, stride.
// w[i] = w[i] + (-rate) * g[i], rounded once, in place.
var sgdRoundedKernel = &kernel{name: "rounded-sgd", nRead: 1, nWrite: 1, body: `
struct Params { length: u32, neg_rate: u32, unused: u32, stride: u32, }
@group(0) @binding(0) var<storage, read> g: array<u32>;
@group(0) @binding(1) var<storage, read_write> w: array<u32>;
@group(0) @binding(2) var<uniform> p: Params;

@compute @workgroup_size(64)
fn main(@builtin(global_invocation_id) gid: vec3<u32>) {
    let i = gid.x + gid.y * p.stride;
    if (i >= p.length) {
        return;
    }
    var r: ExactRegister;
    exact_clear(&r);
    exact_add_product(&r, w[i], 0x3f800000u);
    exact_add_product(&r, p.neg_rate, g[i]);
    w[i] = exact_round(r);
}
`}

// elementwiseGroups spreads length invocations over a two-dimensional grid
// so a long vector stays within the per-dimension dispatch limit, and returns
// the grid and the row stride in invocations.
func elementwiseGroups(length int) (groupsX, groupsY, stride uint32) {
	groups := (length + 63) / 64
	if groups < 1 {
		groups = 1
	}
	const maxX = 32768
	x := groups
	if x > maxX {
		x = maxX
	}
	y := (groups + x - 1) / x
	return uint32(x), uint32(y), uint32(x) * 64
}

// MatMulRounded computes c = op(a) x op(b) where op transposes when asked.
// a and b are row-major as stored: a holds [m,k] (or [k,m] when transA) and b
// holds [k,n] (or [n,k] when transB).
func MatMulRounded(a, b *Buffer, m, k, n int, transA, transB bool) (*Buffer, error) {
	if m < 0 || k < 0 || n < 0 {
		return nil, fmt.Errorf("wgpu: rounded matmul dimensions must not be negative, got [%d,%d]x[%d,%d]", m, k, k, n)
	}
	if a.Len() != m*k || b.Len() != k*n {
		return nil, fmt.Errorf("wgpu: rounded matmul buffers hold %d and %d values, want %d and %d", a.Len(), b.Len(), m*k, k*n)
	}
	out, err := NewBuffer(m * n)
	if err != nil {
		return nil, err
	}
	if m == 0 || n == 0 {
		return out, nil
	}
	var flags uint32
	if transA {
		flags |= 1
	}
	if transB {
		flags |= 2
	}
	if err := matMulRoundedKernel.run([]*Buffer{a, b}, []*Buffer{out}, [16]uint32{uint32(m), uint32(k), uint32(n), flags},
		uint32((n+15)/16), uint32((m+15)/16)); err != nil {
		out.Release()
		return nil, err
	}
	return out, nil
}

// AddRounded computes out[i] = a[i] + b[i mod b.Len()], so a same-length b is
// an elementwise sum and a shorter b broadcasts over the trailing dimension.
func AddRounded(a, b *Buffer) (*Buffer, error) {
	if b.Len() == 0 || a.Len()%b.Len() != 0 {
		return nil, fmt.Errorf("wgpu: rounded add cannot broadcast %d values over %d", b.Len(), a.Len())
	}
	out, err := NewBuffer(a.Len())
	if err != nil {
		return nil, err
	}
	gx, gy, stride := elementwiseGroups(a.Len())
	if err := addRoundedKernel.run([]*Buffer{a, b}, []*Buffer{out}, [16]uint32{uint32(a.Len()), uint32(b.Len()), 0, stride}, gx, gy); err != nil {
		out.Release()
		return nil, err
	}
	return out, nil
}

// SumRowsRounded reduces x, shaped [m,n], over its rows to n values.
func SumRowsRounded(x *Buffer, m, n int) (*Buffer, error) {
	if m < 0 || n < 0 || x.Len() != m*n {
		return nil, fmt.Errorf("wgpu: rounded row sum buffer holds %d values, want %d", x.Len(), m*n)
	}
	out, err := NewBuffer(n)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return out, nil
	}
	if err := sumRowsRoundedKernel.run([]*Buffer{x}, []*Buffer{out}, [16]uint32{uint32(m), uint32(n), 0, 0}, uint32((n+63)/64), 1); err != nil {
		out.Release()
		return nil, err
	}
	return out, nil
}

// SGDRounded updates w in place: w = w - rate*g, rounded once.
func SGDRounded(w, g *Buffer, negatedRateBits uint32) error {
	if w.Len() != g.Len() {
		return fmt.Errorf("wgpu: rounded sgd parameter holds %d values and gradient %d", w.Len(), g.Len())
	}
	if w.Len() == 0 {
		return nil
	}
	gx, gy, stride := elementwiseGroups(w.Len())
	return sgdRoundedKernel.run([]*Buffer{g}, []*Buffer{w}, [16]uint32{uint32(w.Len()), negatedRateBits, 0, stride}, gx, gy)
}

// Clone copies a buffer on the device without a host round trip.
func Clone(source *Buffer) (*Buffer, error) {
	out, err := NewBuffer(source.Len())
	if err != nil {
		return nil, err
	}
	if source.Len() == 0 {
		return out, nil
	}
	submitMu.Lock()
	defer submitMu.Unlock()
	h, err := acquire()
	if err != nil {
		out.Release()
		return nil, fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	source.mu.Lock()
	defer source.mu.Unlock()
	out.mu.Lock()
	defer out.mu.Unlock()
	if source.released || source.buffer == nil {
		out.released = true
		return nil, fmt.Errorf("wgpu: clone of a released buffer")
	}
	encoder, err := h.device.CreateCommandEncoder(nil)
	if err != nil {
		return nil, fmt.Errorf("wgpu: create clone encoder: %w", err)
	}
	encoder.CopyBufferToBuffer(source.buffer, 0, out.buffer, 0, bufferBytes(source.Len()))
	commands, err := encoder.Finish()
	if err != nil {
		return nil, fmt.Errorf("wgpu: finish clone encoder: %w", err)
	}
	if _, err := h.device.Queue().Submit(commands); err != nil {
		return nil, fmt.Errorf("wgpu: submit clone: %w", err)
	}
	return out, nil
}

// divRoundedKernel: params = length, 0, 0, stride. out[i] = a[i] / b[i].
var divRoundedKernel = &kernel{name: "rounded-div", nRead: 2, nWrite: 1, body: `
struct Params { length: u32, unused0: u32, unused1: u32, stride: u32, }
@group(0) @binding(0) var<storage, read> a: array<u32>;
@group(0) @binding(1) var<storage, read> b: array<u32>;
@group(0) @binding(2) var<storage, read_write> c: array<u32>;
@group(0) @binding(3) var<uniform> p: Params;

@compute @workgroup_size(64)
fn main(@builtin(global_invocation_id) gid: vec3<u32>) {
    let i = gid.x + gid.y * p.stride;
    if (i >= p.length) {
        return;
    }
    c[i] = f32_div(a[i], b[i]);
}
`}

// sqrtRoundedKernel: params = length, 0, 0, stride. out[i] = sqrt(a[i]).
var sqrtRoundedKernel = &kernel{name: "rounded-sqrt", nRead: 1, nWrite: 1, body: `
struct Params { length: u32, unused0: u32, unused1: u32, stride: u32, }
@group(0) @binding(0) var<storage, read> a: array<u32>;
@group(0) @binding(1) var<storage, read_write> c: array<u32>;
@group(0) @binding(2) var<uniform> p: Params;

@compute @workgroup_size(64)
fn main(@builtin(global_invocation_id) gid: vec3<u32>) {
    let i = gid.x + gid.y * p.stride;
    if (i >= p.length) {
        return;
    }
    c[i] = f32_sqrt(a[i]);
}
`}

// DivRounded divides a by b elementwise, each quotient correctly rounded.
func DivRounded(a, b *Buffer) (*Buffer, error) {
	if a.Len() != b.Len() {
		return nil, fmt.Errorf("wgpu: rounded div buffers hold %d and %d values", a.Len(), b.Len())
	}
	out, err := NewBuffer(a.Len())
	if err != nil {
		return nil, err
	}
	if a.Len() == 0 {
		return out, nil
	}
	gx, gy, stride := elementwiseGroups(a.Len())
	if err := divRoundedKernel.run([]*Buffer{a, b}, []*Buffer{out}, [16]uint32{uint32(a.Len()), 0, 0, stride}, gx, gy); err != nil {
		out.Release()
		return nil, err
	}
	return out, nil
}

// SqrtRounded takes the square root of each element, correctly rounded.
func SqrtRounded(a *Buffer) (*Buffer, error) {
	out, err := NewBuffer(a.Len())
	if err != nil {
		return nil, err
	}
	if a.Len() == 0 {
		return out, nil
	}
	gx, gy, stride := elementwiseGroups(a.Len())
	if err := sqrtRoundedKernel.run([]*Buffer{a}, []*Buffer{out}, [16]uint32{uint32(a.Len()), 0, 0, stride}, gx, gy); err != nil {
		out.Release()
		return nil, err
	}
	return out, nil
}

// AdamScalars are the float32 constants of one Adam or AdamW step, each
// computed once on the host so the host and the device read the same bits.
type AdamScalars struct {
	Beta1, OneMinusBeta1, Beta2, OneMinusBeta2 float32
	Bias1, Bias2, Epsilon                      float32
	// NegRate and NegDecay are -lr and -(lr*weightDecay); Decay says whether
	// the decoupled decay of AdamW applies.
	NegRate, NegDecay float32
	Decay             bool
}

// adamRoundedKernel: params = length, flags, 0, stride, then the scalars.
// Every step is one correctly rounded operation or one exact two-term sum
// rounded once; see nn's adam_rounded.go for the definition.
var adamRoundedKernel = &kernel{name: "rounded-adam", nRead: 1, nWrite: 3, body: `
struct Params {
    length: u32, flags: u32, unused: u32, stride: u32,
    b1: u32, c1: u32, b2: u32, c2: u32,
    bias1: u32, bias2: u32, eps: u32, neg_rate: u32,
    neg_decay: u32, pad0: u32, pad1: u32, pad2: u32,
}
@group(0) @binding(0) var<storage, read> g: array<u32>;
@group(0) @binding(1) var<storage, read_write> w: array<u32>;
@group(0) @binding(2) var<storage, read_write> m: array<u32>;
@group(0) @binding(3) var<storage, read_write> v: array<u32>;
@group(0) @binding(4) var<uniform> p: Params;

fn sum2(a: u32, b: u32, c: u32, d: u32) -> u32 {
    var r: ExactRegister;
    exact_clear(&r);
    exact_add_product(&r, a, b);
    exact_add_product(&r, c, d);
    return exact_round(r);
}

@compute @workgroup_size(64)
fn main(@builtin(global_invocation_id) gid: vec3<u32>) {
    let i = gid.x + gid.y * p.stride;
    if (i >= p.length) {
        return;
    }
    let one = 0x3f800000u;
    var wv = w[i];
    if ((p.flags & 1u) != 0u) {
        wv = sum2(wv, one, p.neg_decay, wv);
    }
    let gv = g[i];
    let mn = sum2(p.b1, m[i], p.c1, gv);
    var rg: ExactRegister;
    exact_clear(&rg);
    exact_add_product(&rg, gv, gv);
    let g2 = exact_round(rg);
    let vn = sum2(p.b2, v[i], p.c2, g2);
    let mhat = f32_div(mn, p.bias1);
    let vhat = f32_div(vn, p.bias2);
    let den = sum2(f32_sqrt(vhat), one, p.eps, one);
    let q = f32_div(mhat, den);
    w[i] = sum2(wv, one, p.neg_rate, q);
    m[i] = mn;
    v[i] = vn;
}
`}

// AdamRounded applies one Adam or AdamW step in place to w, m and v from the
// gradient g.
func AdamRounded(w, g, m, v *Buffer, s AdamScalars) error {
	n := w.Len()
	if g.Len() != n || m.Len() != n || v.Len() != n {
		return fmt.Errorf("wgpu: rounded adam buffers hold %d, %d, %d and %d values", n, g.Len(), m.Len(), v.Len())
	}
	if n == 0 {
		return nil
	}
	var flags uint32
	if s.Decay {
		flags = 1
	}
	gx, gy, stride := elementwiseGroups(n)
	bits := math.Float32bits
	return adamRoundedKernel.run([]*Buffer{g}, []*Buffer{w, m, v}, [16]uint32{
		uint32(n), flags, 0, stride,
		bits(s.Beta1), bits(s.OneMinusBeta1), bits(s.Beta2), bits(s.OneMinusBeta2),
		bits(s.Bias1), bits(s.Bias2), bits(s.Epsilon), bits(s.NegRate),
		bits(s.NegDecay), 0, 0, 0,
	}, gx, gy)
}

// expRoundedKernel and logRoundedKernel: params = length, 0, 0, stride.
var expRoundedKernel = &kernel{name: "rounded-exp", nRead: 2, nWrite: 1, fixed: true, body: `
struct Params { length: u32, unused0: u32, unused1: u32, stride: u32, }
@group(0) @binding(0) var<storage, read> a: array<u32>;
@group(0) @binding(1) var<storage, read> consts: array<u32>;
@group(0) @binding(2) var<storage, read_write> c: array<u32>;
@group(0) @binding(3) var<uniform> p: Params;

@compute @workgroup_size(64)
fn main(@builtin(global_invocation_id) gid: vec3<u32>) {
    let i = gid.x + gid.y * p.stride;
    if (i >= p.length) {
        return;
    }
    c[i] = exp_rounded(a[i]);
}
`}

var logRoundedKernel = &kernel{name: "rounded-log", nRead: 2, nWrite: 1, fixed: true, body: `
struct Params { length: u32, unused0: u32, unused1: u32, stride: u32, }
@group(0) @binding(0) var<storage, read> a: array<u32>;
@group(0) @binding(1) var<storage, read> consts: array<u32>;
@group(0) @binding(2) var<storage, read_write> c: array<u32>;
@group(0) @binding(3) var<uniform> p: Params;

@compute @workgroup_size(64)
fn main(@builtin(global_invocation_id) gid: vec3<u32>) {
    let i = gid.x + gid.y * p.stride;
    if (i >= p.length) {
        return;
    }
    c[i] = log_rounded(a[i]);
}
`}

// ExpRounded computes e^x for each element, correctly rounded.
func ExpRounded(a *Buffer) (*Buffer, error) { return unaryRounded(expRoundedKernel, a) }

// LogRounded computes the natural logarithm of each element, correctly rounded.
func LogRounded(a *Buffer) (*Buffer, error) { return unaryRounded(logRoundedKernel, a) }

func unaryRounded(k *kernel, a *Buffer) (*Buffer, error) {
	out, err := NewBuffer(a.Len())
	if err != nil {
		return nil, err
	}
	if a.Len() == 0 {
		return out, nil
	}
	consts, err := fixedConstants()
	if err != nil {
		out.Release()
		return nil, err
	}
	gx, gy, stride := elementwiseGroups(a.Len())
	if err := k.run([]*Buffer{a, consts}, []*Buffer{out}, [16]uint32{uint32(a.Len()), 0, 0, stride}, gx, gy); err != nil {
		out.Release()
		return nil, err
	}
	return out, nil
}
