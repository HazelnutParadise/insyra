package wgpu

import "fmt"

// Soft-target cross-entropy and distillation in PrecisionFloat32Rounded. Each
// kernel is one line of the sequence nn's soft_ce_rounded.go defines, one
// correctly rounded operation or an exact sum rounded once, so the device
// returns the host's bits. Every kernel here uses the same eight-word
// parameter block: n, classes, rows as float32 bits, stride, mode.

const sceSoftCommon = sceCommon + `
struct Params { n: u32, classes: u32, rows_bits: u32, stride: u32, mode: u32, u0: u32, u1: u32, u2: u32, }

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

fn sce_neg(bits: u32) -> u32 {
    return bits ^ 0x80000000u;
}

// The peak's entry of u: 0, or the probability itself when it is NaN.
fn sce_peak_u(q: u32) -> u32 {
    if ((q & 0x7fffffffu) > 0x7f800000u) {
        return q;
    }
    return 0u;
}

fn sce_mul(a: u32, b: u32) -> u32 {
    var r: ExactRegister;
    exact_clear(&r);
    exact_add_product(&r, a, b);
    return exact_round(r);
}
`

// sceStatsKernel, per row: peak is the first index of the largest logit, oth
// the exact sum of the other exponentials rounded once, sum the exact sum of
// all of them, ell = log1p(oth).
var sceStatsKernel = &kernel{name: "rounded-soft-stats", nRead: 3, nWrite: 4, fixed: true, body: sceSoftCommon + `
@group(0) @binding(0) var<storage, read> x: array<u32>;
@group(0) @binding(1) var<storage, read> ee: array<u32>;
@group(0) @binding(2) var<storage, read> consts: array<u32>;
@group(0) @binding(3) var<storage, read_write> peak: array<u32>;
@group(0) @binding(4) var<storage, read_write> oth: array<u32>;
@group(0) @binding(5) var<storage, read_write> sum: array<u32>;
@group(0) @binding(6) var<storage, read_write> ell: array<u32>;
@group(0) @binding(7) var<uniform> p: Params;

@compute @workgroup_size(64)
fn main(@builtin(global_invocation_id) gid: vec3<u32>) {
    let row = gid.x + gid.y * p.stride;
    if (row >= p.n) {
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
    let o = exact_round(others);
    peak[row] = best;
    oth[row] = o;
    sum[row] = exact_round(total);
    ell[row] = log1p_rounded(o);
}
`}

// sceDivKernel, per element: out = ee / sum[row], the probabilities.
var sceDivKernel = &kernel{name: "rounded-soft-prob", nRead: 2, nWrite: 1, body: sceSoftCommon + `
@group(0) @binding(0) var<storage, read> ee: array<u32>;
@group(0) @binding(1) var<storage, read> sum: array<u32>;
@group(0) @binding(2) var<storage, read_write> q: array<u32>;
@group(0) @binding(3) var<uniform> p: Params;

@compute @workgroup_size(64)
fn main(@builtin(global_invocation_id) gid: vec3<u32>) {
    let i = gid.x + gid.y * p.stride;
    if (i >= p.n) {
        return;
    }
    q[i] = f32_div(ee[i], sum[i / p.classes]);
}
`}

// sceSoftLossKernel, per row: the exact sum of pr*ell - pr*d, rounded once.
var sceSoftLossKernel = &kernel{name: "rounded-soft-loss", nRead: 3, nWrite: 1, body: sceSoftCommon + `
@group(0) @binding(0) var<storage, read> pr: array<u32>;
@group(0) @binding(1) var<storage, read> dd: array<u32>;
@group(0) @binding(2) var<storage, read> ell: array<u32>;
@group(0) @binding(3) var<storage, read_write> loss: array<u32>;
@group(0) @binding(4) var<uniform> p: Params;

@compute @workgroup_size(64)
fn main(@builtin(global_invocation_id) gid: vec3<u32>) {
    let row = gid.x + gid.y * p.stride;
    if (row >= p.n) {
        return;
    }
    let base = row * p.classes;
    var r: ExactRegister;
    exact_clear(&r);
    for (var c: u32 = 0u; c < p.classes; c = c + 1u) {
        // A class with probability 0 adds nothing, even when its centred
        // difference overflowed to -Inf and 0 * Inf would be NaN.
        if ((pr[base + c] & 0x7fffffffu) == 0u) {
            continue;
        }
        exact_add_product(&r, pr[base + c], ell[row]);
        exact_add_product(&r, pr[base + c], sce_neg(dd[base + c]));
    }
    loss[row] = exact_round(r);
}
`}

// sceMassKernel, per row: the exact sum of the target probabilities, rounded once.
var sceMassKernel = &kernel{name: "rounded-soft-mass", nRead: 1, nWrite: 1, body: sceSoftCommon + `
@group(0) @binding(0) var<storage, read> pr: array<u32>;
@group(0) @binding(1) var<storage, read_write> mass: array<u32>;
@group(0) @binding(2) var<uniform> p: Params;

@compute @workgroup_size(64)
fn main(@builtin(global_invocation_id) gid: vec3<u32>) {
    let row = gid.x + gid.y * p.stride;
    if (row >= p.n) {
        return;
    }
    var r: ExactRegister;
    exact_clear(&r);
    for (var c: u32 = 0u; c < p.classes; c = c + 1u) {
        exact_add_product(&r, pr[row * p.classes + c], ONE);
    }
    mass[row] = exact_round(r);
}
`}

// sceSoftUKernel, per element: u = q*mass - pr rounded once, 0 at the peak.
// mode 1 takes mass as 1.
var sceSoftUKernel = &kernel{name: "rounded-soft-u", nRead: 4, nWrite: 1, body: sceSoftCommon + `
@group(0) @binding(0) var<storage, read> q: array<u32>;
@group(0) @binding(1) var<storage, read> pr: array<u32>;
@group(0) @binding(2) var<storage, read> peak: array<u32>;
@group(0) @binding(3) var<storage, read> mass: array<u32>;
@group(0) @binding(4) var<storage, read_write> u: array<u32>;
@group(0) @binding(5) var<uniform> p: Params;

@compute @workgroup_size(64)
fn main(@builtin(global_invocation_id) gid: vec3<u32>) {
    let i = gid.x + gid.y * p.stride;
    if (i >= p.n) {
        return;
    }
    let row = i / p.classes;
    if (i % p.classes == peak[row]) {
        // 0 for a finite row; a NaN probability (a non-finite logit in the
        // row) stays NaN, so a one-class row does not hide it.
        u[i] = sce_peak_u(q[i]);
        return;
    }
    var m = mass[row];
    if (p.mode == 1u) {
        m = ONE;
    }
    u[i] = sce_sum2(q[i], m, sce_neg(pr[i]), ONE);
}
`}

// sceTotalKernel, per row: the exact sum of u, rounded once.
var sceTotalKernel = &kernel{name: "rounded-soft-total", nRead: 1, nWrite: 1, body: sceSoftCommon + `
@group(0) @binding(0) var<storage, read> u: array<u32>;
@group(0) @binding(1) var<storage, read_write> tot: array<u32>;
@group(0) @binding(2) var<uniform> p: Params;

@compute @workgroup_size(64)
fn main(@builtin(global_invocation_id) gid: vec3<u32>) {
    let row = gid.x + gid.y * p.stride;
    if (row >= p.n) {
        return;
    }
    var r: ExactRegister;
    exact_clear(&r);
    for (var c: u32 = 0u; c < p.classes; c = c + 1u) {
        exact_add_product(&r, u[row * p.classes + c], ONE);
    }
    tot[row] = exact_round(r);
}
`}

// sceSoftGradKernel, per element: scale = upstream / rows. The peak's gradient
// is minus the total of the others times scale, the rest u times scale; in
// mode 1 a row whose fb is 0 takes (u - q*total) times scale instead.
var sceSoftGradKernel = &kernel{name: "rounded-soft-grad", nRead: 6, nWrite: 1, body: sceSoftCommon + `
@group(0) @binding(0) var<storage, read> u: array<u32>;
@group(0) @binding(1) var<storage, read> q: array<u32>;
@group(0) @binding(2) var<storage, read> peak: array<u32>;
@group(0) @binding(3) var<storage, read> tot: array<u32>;
@group(0) @binding(4) var<storage, read> fb: array<u32>;
@group(0) @binding(5) var<storage, read> upstream: array<u32>;
@group(0) @binding(6) var<storage, read_write> grad: array<u32>;
@group(0) @binding(7) var<uniform> p: Params;

@compute @workgroup_size(64)
fn main(@builtin(global_invocation_id) gid: vec3<u32>) {
    let i = gid.x + gid.y * p.stride;
    if (i >= p.n) {
        return;
    }
    let row = i / p.classes;
    let scale = f32_div(upstream[0], p.rows_bits);
    if (p.mode == 1u && fb[row] == 0u) {
        grad[i] = sce_mul(sce_sum2(u[i], ONE, sce_neg(q[i]), tot[row]), scale);
        return;
    }
    if (i % p.classes == peak[row]) {
        grad[i] = sce_mul(sce_neg(tot[row]), scale);
        return;
    }
    grad[i] = sce_mul(u[i], scale);
}
`}

// sceDeltaKernel, per element: zS - zS[0] - zT + zT[0], exact, rounded once.
var sceDeltaKernel = &kernel{name: "rounded-distill-delta", nRead: 2, nWrite: 1, body: sceSoftCommon + `
@group(0) @binding(0) var<storage, read> zs: array<u32>;
@group(0) @binding(1) var<storage, read> zt: array<u32>;
@group(0) @binding(2) var<storage, read_write> delta: array<u32>;
@group(0) @binding(3) var<uniform> p: Params;

@compute @workgroup_size(64)
fn main(@builtin(global_invocation_id) gid: vec3<u32>) {
    let i = gid.x + gid.y * p.stride;
    if (i >= p.n) {
        return;
    }
    let base = (i / p.classes) * p.classes;
    var r: ExactRegister;
    exact_clear(&r);
    exact_add_product(&r, zs[i], ONE);
    exact_add_product(&r, zs[base], NEG_ONE);
    exact_add_product(&r, zt[i], NEG_ONE);
    exact_add_product(&r, zt[base], ONE);
    delta[i] = exact_round(r);
}
`}

// sceDistMeanKernel, per row: mu = the exact sum of pt*delta rounded once, and
// zero = 1 when some teacher probability is not above 0.
var sceDistMeanKernel = &kernel{name: "rounded-distill-mean", nRead: 2, nWrite: 2, body: sceSoftCommon + `
@group(0) @binding(0) var<storage, read> pt: array<u32>;
@group(0) @binding(1) var<storage, read> delta: array<u32>;
@group(0) @binding(2) var<storage, read_write> mu: array<u32>;
@group(0) @binding(3) var<storage, read_write> zero: array<u32>;
@group(0) @binding(4) var<uniform> p: Params;

@compute @workgroup_size(64)
fn main(@builtin(global_invocation_id) gid: vec3<u32>) {
    let row = gid.x + gid.y * p.stride;
    if (row >= p.n) {
        return;
    }
    let base = row * p.classes;
    var r: ExactRegister;
    exact_clear(&r);
    var any_zero: u32 = 0u;
    for (var c: u32 = 0u; c < p.classes; c = c + 1u) {
        let b = pt[base + c];
        if ((b >> 31u) == 1u || b == 0u || b > 0x7f800000u) {
            any_zero = 1u;
        }
        exact_add_product(&r, b, delta[base + c]);
    }
    mu[row] = exact_round(r);
    zero[row] = any_zero;
}
`}

// sceDistCentreKernel, per element: d = delta - mu rounded once and expm1(d).
var sceDistCentreKernel = &kernel{name: "rounded-distill-centre", nRead: 3, nWrite: 2, fixed: true, body: sceSoftCommon + `
@group(0) @binding(0) var<storage, read> delta: array<u32>;
@group(0) @binding(1) var<storage, read> mu: array<u32>;
@group(0) @binding(2) var<storage, read> consts: array<u32>;
@group(0) @binding(3) var<storage, read_write> dd: array<u32>;
@group(0) @binding(4) var<storage, read_write> ex: array<u32>;
@group(0) @binding(5) var<uniform> p: Params;

@compute @workgroup_size(64)
fn main(@builtin(global_invocation_id) gid: vec3<u32>) {
    let i = gid.x + gid.y * p.stride;
    if (i >= p.n) {
        return;
    }
    let d = sce_sum2(delta[i], ONE, NEG_ONE, mu[i / p.classes]);
    dd[i] = d;
    ex[i] = expm1_rounded(d);
}
`}

// sceDistShiftKernel, per row: sigma = log1p(sum(pt*expm1(d))) and fb = 1 when
// the row falls back to q - p (a teacher probability not above 0, or some |d|
// not at most 1).
var sceDistShiftKernel = &kernel{name: "rounded-distill-shift", nRead: 5, nWrite: 2, fixed: true, body: sceSoftCommon + `
@group(0) @binding(0) var<storage, read> pt: array<u32>;
@group(0) @binding(1) var<storage, read> dd: array<u32>;
@group(0) @binding(2) var<storage, read> ex: array<u32>;
@group(0) @binding(3) var<storage, read> zero: array<u32>;
@group(0) @binding(4) var<storage, read> consts: array<u32>;
@group(0) @binding(5) var<storage, read_write> sigma: array<u32>;
@group(0) @binding(6) var<storage, read_write> fb: array<u32>;
@group(0) @binding(7) var<uniform> p: Params;

@compute @workgroup_size(64)
fn main(@builtin(global_invocation_id) gid: vec3<u32>) {
    let row = gid.x + gid.y * p.stride;
    if (row >= p.n) {
        return;
    }
    let base = row * p.classes;
    var r: ExactRegister;
    exact_clear(&r);
    var bad = zero[row];
    for (var c: u32 = 0u; c < p.classes; c = c + 1u) {
        let a = dd[base + c] & 0x7fffffffu;
        if (a > 0x3f800000u) {
            bad = 1u;
        }
        exact_add_product(&r, pt[base + c], ex[base + c]);
    }
    sigma[row] = log1p_rounded(exact_round(r));
    fb[row] = bad;
}
`}

// sceDistUKernel, per element: u = pt*expm1(d - sigma) rounded, or for a
// fallback row q - pt rounded once with 0 at the peak.
var sceDistUKernel = &kernel{name: "rounded-distill-u", nRead: 7, nWrite: 1, fixed: true, body: sceSoftCommon + `
@group(0) @binding(0) var<storage, read> pt: array<u32>;
@group(0) @binding(1) var<storage, read> q: array<u32>;
@group(0) @binding(2) var<storage, read> dd: array<u32>;
@group(0) @binding(3) var<storage, read> sigma: array<u32>;
@group(0) @binding(4) var<storage, read> fb: array<u32>;
@group(0) @binding(5) var<storage, read> peak: array<u32>;
@group(0) @binding(6) var<storage, read> consts: array<u32>;
@group(0) @binding(7) var<storage, read_write> u: array<u32>;
@group(0) @binding(8) var<uniform> p: Params;

@compute @workgroup_size(64)
fn main(@builtin(global_invocation_id) gid: vec3<u32>) {
    let i = gid.x + gid.y * p.stride;
    if (i >= p.n) {
        return;
    }
    let row = i / p.classes;
    if (fb[row] == 1u) {
        if (i % p.classes == peak[row]) {
            u[i] = sce_peak_u(q[i]);
        } else {
            u[i] = sce_sum2(q[i], ONE, sce_neg(pt[i]), ONE);
        }
        return;
    }
    let s = sce_sum2(dd[i], ONE, NEG_ONE, sigma[row]);
    u[i] = sce_mul(pt[i], expm1_rounded(s));
}
`}

// softState holds what a soft-target backward pass reuses.
type softState struct {
	Q, Peak, P *Buffer // P is the teacher's probabilities for distillation, else nil
}

// Release frees the state.
func (s *softState) Release() {
	if s == nil {
		return
	}
	for _, b := range []*Buffer{s.Q, s.Peak, s.P} {
		if b != nil {
			b.Release()
		}
	}
}

// SoftState is the exported name of softState for the accel package.
type SoftState = softState

type scratchSet struct{ buffers []*Buffer }

func (s *scratchSet) alloc(n int) (*Buffer, error) {
	b, err := NewBuffer(n)
	if err == nil {
		s.buffers = append(s.buffers, b)
	}
	return b, err
}

func (s *scratchSet) release() {
	for _, b := range s.buffers {
		b.Release()
	}
	s.buffers = nil
}

// keep takes b out of the set so release does not free it.
func (s *scratchSet) keep(b *Buffer) {
	for i, x := range s.buffers {
		if x == b {
			s.buffers = append(s.buffers[:i], s.buffers[i+1:]...)
			return
		}
	}
}

func softParams(n, classes int, rowsBits, stride, mode uint32) [16]uint32 {
	return [16]uint32{uint32(n), uint32(classes), rowsBits, stride, mode}
}

// rowsAndElements runs k once per row (perRow) or per element.
func runSoft(k *kernel, reads, writes []*Buffer, rows, classes int, perRow bool, rowsBits, mode uint32) error {
	n := rows * classes
	if perRow {
		n = rows
	}
	gx, gy, stride := elementwiseGroups(n)
	return k.run(reads, writes, softParams(n, classes, rowsBits, stride, mode), gx, gy)
}

// softmaxStats runs max, exp and the row statistics on logits [rows,classes]
// and returns the per-element d and probabilities plus the per-row peak and
// ell. d and q and peak are the caller's to release through the set.
func softmaxStats(s *scratchSet, consts, logits *Buffer, rows, classes int) (d, q, peak, ell *Buffer, err error) {
	var maxes, e, oth, sum *Buffer
	for _, dst := range []struct {
		out **Buffer
		n   int
	}{{&maxes, rows}, {&d, rows * classes}, {&e, rows * classes}, {&peak, rows}, {&oth, rows}, {&sum, rows}, {&ell, rows}, {&q, rows * classes}} {
		if *dst.out, err = s.alloc(dst.n); err != nil {
			return
		}
	}
	rg, ry, rstride := elementwiseGroups(rows)
	eg, ey, estride := elementwiseGroups(rows * classes)
	if err = sceMaxKernel.run([]*Buffer{logits}, []*Buffer{maxes}, [16]uint32{uint32(rows), uint32(classes), 0, rstride}, rg, ry); err != nil {
		return
	}
	if err = sceExpKernel.run([]*Buffer{logits, maxes, consts}, []*Buffer{d, e}, [16]uint32{uint32(rows * classes), uint32(classes), 0, estride}, eg, ey); err != nil {
		return
	}
	if err = runSoft(sceStatsKernel, []*Buffer{logits, e, consts}, []*Buffer{peak, oth, sum, ell}, rows, classes, true, 0, 0); err != nil {
		return
	}
	err = runSoft(sceDivKernel, []*Buffer{e, sum}, []*Buffer{q}, rows, classes, false, 0, 0)
	return
}

func meanOfLosses(s *scratchSet, losses *Buffer, rows int) (*Buffer, error) {
	out, err := NewBuffer(1)
	if err != nil {
		return nil, err
	}
	if err := sceMeanKernel.run([]*Buffer{losses}, []*Buffer{out}, [16]uint32{uint32(rows), ceBits(float32(rows)), 0, 0}, 1, 1); err != nil {
		out.Release()
		return nil, err
	}
	return out, nil
}

func checkSoftShapes(op string, a, b *Buffer, rows, classes int) error {
	if rows <= 0 || classes <= 0 || a.Len() != rows*classes || b.Len() != rows*classes {
		return fmt.Errorf("wgpu: %s buffers hold %d and %d values for [%d,%d]", op, a.Len(), b.Len(), rows, classes)
	}
	return nil
}

// SoftTargetCrossEntropyRounded returns the mean loss of logits [rows,classes]
// against probability rows, with the state the gradient needs.
func SoftTargetCrossEntropyRounded(logits, targets *Buffer, rows, classes int) (*Buffer, *SoftState, error) {
	if err := checkSoftShapes("soft-target cross-entropy", logits, targets, rows, classes); err != nil {
		return nil, nil, err
	}
	consts, err := fixedConstants()
	if err != nil {
		return nil, nil, err
	}
	var s scratchSet
	defer s.release()
	d, q, peak, ell, err := softmaxStats(&s, consts, logits, rows, classes)
	if err != nil {
		return nil, nil, err
	}
	losses, err := s.alloc(rows)
	if err != nil {
		return nil, nil, err
	}
	if err := runSoft(sceSoftLossKernel, []*Buffer{targets, d, ell}, []*Buffer{losses}, rows, classes, true, 0, 0); err != nil {
		return nil, nil, err
	}
	out, err := meanOfLosses(&s, losses, rows)
	if err != nil {
		return nil, nil, err
	}
	s.keep(q)
	s.keep(peak)
	return out, &SoftState{Q: q, Peak: peak}, nil
}

// SoftTargetCrossEntropyGradRounded returns the gradient with respect to the logits.
func SoftTargetCrossEntropyGradRounded(state *SoftState, targets, upstream *Buffer, rows, classes int) (*Buffer, error) {
	if state == nil || state.Q.Len() != rows*classes || targets.Len() != rows*classes || upstream.Len() != 1 {
		return nil, fmt.Errorf("wgpu: soft-target gradient buffers do not match [%d,%d]", rows, classes)
	}
	var s scratchSet
	defer s.release()
	mass, err := s.alloc(rows)
	if err != nil {
		return nil, err
	}
	u, err := s.alloc(rows * classes)
	if err != nil {
		return nil, err
	}
	tot, err := s.alloc(rows)
	if err != nil {
		return nil, err
	}
	if err := runSoft(sceMassKernel, []*Buffer{targets}, []*Buffer{mass}, rows, classes, true, 0, 0); err != nil {
		return nil, err
	}
	if err := runSoft(sceSoftUKernel, []*Buffer{state.Q, targets, state.Peak, mass}, []*Buffer{u}, rows, classes, false, 0, 0); err != nil {
		return nil, err
	}
	if err := runSoft(sceTotalKernel, []*Buffer{u}, []*Buffer{tot}, rows, classes, true, 0, 0); err != nil {
		return nil, err
	}
	out, err := NewBuffer(rows * classes)
	if err != nil {
		return nil, err
	}
	if err := runSoft(sceSoftGradKernel, []*Buffer{u, state.Q, state.Peak, tot, tot, upstream}, []*Buffer{out}, rows, classes, false, ceBits(float32(rows)), 0); err != nil {
		out.Release()
		return nil, err
	}
	return out, nil
}

// DistillationRounded returns the mean loss of the student's logits against
// softmax(teacher), with the state the gradient needs.
func DistillationRounded(student, teacher *Buffer, rows, classes int) (*Buffer, *SoftState, error) {
	if err := checkSoftShapes("distillation", student, teacher, rows, classes); err != nil {
		return nil, nil, err
	}
	consts, err := fixedConstants()
	if err != nil {
		return nil, nil, err
	}
	var s scratchSet
	defer s.release()
	d, q, peak, ell, err := softmaxStats(&s, consts, student, rows, classes)
	if err != nil {
		return nil, nil, err
	}
	_, pt, _, _, err := softmaxStats(&s, consts, teacher, rows, classes)
	if err != nil {
		return nil, nil, err
	}
	losses, err := s.alloc(rows)
	if err != nil {
		return nil, nil, err
	}
	if err := runSoft(sceSoftLossKernel, []*Buffer{pt, d, ell}, []*Buffer{losses}, rows, classes, true, 0, 0); err != nil {
		return nil, nil, err
	}
	out, err := meanOfLosses(&s, losses, rows)
	if err != nil {
		return nil, nil, err
	}
	s.keep(q)
	s.keep(peak)
	s.keep(pt)
	return out, &SoftState{Q: q, Peak: peak, P: pt}, nil
}

// DistillationGradRounded returns the gradient with respect to the student's logits.
func DistillationGradRounded(state *SoftState, student, teacher, upstream *Buffer, rows, classes int) (*Buffer, error) {
	if state == nil || state.P == nil || state.Q.Len() != rows*classes || student.Len() != rows*classes || teacher.Len() != rows*classes || upstream.Len() != 1 {
		return nil, fmt.Errorf("wgpu: distillation gradient buffers do not match [%d,%d]", rows, classes)
	}
	consts, err := fixedConstants()
	if err != nil {
		return nil, err
	}
	var s scratchSet
	defer s.release()
	var delta, mu, zero, dd, ex, sigma, fb, u, tot *Buffer
	for _, dst := range []struct {
		out **Buffer
		n   int
	}{{&delta, rows * classes}, {&mu, rows}, {&zero, rows}, {&dd, rows * classes}, {&ex, rows * classes}, {&sigma, rows}, {&fb, rows}, {&u, rows * classes}, {&tot, rows}} {
		if *dst.out, err = s.alloc(dst.n); err != nil {
			return nil, err
		}
	}
	steps := []func() error{
		func() error {
			return runSoft(sceDeltaKernel, []*Buffer{student, teacher}, []*Buffer{delta}, rows, classes, false, 0, 0)
		},
		func() error {
			return runSoft(sceDistMeanKernel, []*Buffer{state.P, delta}, []*Buffer{mu, zero}, rows, classes, true, 0, 0)
		},
		func() error {
			return runSoft(sceDistCentreKernel, []*Buffer{delta, mu, consts}, []*Buffer{dd, ex}, rows, classes, false, 0, 0)
		},
		func() error {
			return runSoft(sceDistShiftKernel, []*Buffer{state.P, dd, ex, zero, consts}, []*Buffer{sigma, fb}, rows, classes, true, 0, 0)
		},
		func() error {
			return runSoft(sceDistUKernel, []*Buffer{state.P, state.Q, dd, sigma, fb, state.Peak, consts}, []*Buffer{u}, rows, classes, false, 0, 0)
		},
		func() error { return runSoft(sceTotalKernel, []*Buffer{u}, []*Buffer{tot}, rows, classes, true, 0, 0) },
	}
	for _, step := range steps {
		if err := step(); err != nil {
			return nil, err
		}
	}
	out, err := NewBuffer(rows * classes)
	if err != nil {
		return nil, err
	}
	if err := runSoft(sceSoftGradKernel, []*Buffer{u, state.Q, state.Peak, tot, fb, upstream}, []*Buffer{out}, rows, classes, false, ceBits(float32(rows)), 1); err != nil {
		out.Release()
		return nil, err
	}
	return out, nil
}
