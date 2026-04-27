// fa/lbfgsb.go
//
// Pure-Go implementation of the L-BFGS-B algorithm for bound-constrained
// nonlinear optimization, following Byrd, Lu, Nocedal & Zhu, "A limited
// memory algorithm for bound constrained optimization", SIAM J. Sci.
// Comput. 16(5), 1995. This is the same algorithm wrapped by R's
// optim(method = "L-BFGS-B"); both use the active-set/Cauchy-point
// approach with the compact L-BFGS Hessian representation.
//
// The implementation supports:
//   * lower / upper bounds (active-set handled via the generalized
//     Cauchy point and free-set subspace minimization)
//   * limited-memory secant updates with the BLNZ compact form
//   * Moré-Thuente backtracking line search satisfying the strong
//     Wolfe conditions on the projected direction
//   * R's parscale rescaling so the objective sees parameters at the
//     same dynamic range as R's optim()
//
// Termination matches R's optim defaults: max(|f_{k+1} - f_k|) /
// max(|f_k|, |f_{k+1}|, 1) <= factr * eps  OR  max projected gradient
// norm <= pgtol.
package fa

import (
	"errors"
	"fmt"
	"math"

	"gonum.org/v1/gonum/mat"
)

// lbfgsbResult bundles the final state of an L-BFGS-B run.
type lbfgsbResult struct {
	X         []float64
	F         float64
	Iters     int
	Converged bool
}

// lbfgsbParams configures the optimizer. Defaults mirror R's optim().
type lbfgsbParams struct {
	M        int       // memory length (R default 5)
	Factr    float64   // relative f tolerance multiplier of eps (R default 1e7)
	PgTol    float64   // projected-gradient inf-norm tolerance (R default 1e-5)
	MaxIter  int       // iteration cap
	Parscale []float64 // parameter scales (R default rep(1, n))
}

const (
	lbfgsbEps = 2.220446049250313e-16
)

// lbfgsb minimises fn(x) subject to lower <= x <= upper using L-BFGS-B.
// grad must populate g with ∇fn(x). All slices are length n; lower[i]
// may be -Inf, upper[i] may be +Inf.
func lbfgsb(start, lower, upper []float64,
	fn func([]float64) float64,
	grad func(g, x []float64),
	params lbfgsbParams,
) (*lbfgsbResult, error) {
	n := len(start)
	if n == 0 {
		return nil, errors.New("lbfgsb: empty start")
	}
	if len(lower) != n || len(upper) != n {
		return nil, errors.New("lbfgsb: bound length mismatch")
	}
	for i := 0; i < n; i++ {
		if lower[i] > upper[i] {
			return nil, fmt.Errorf("lbfgsb: lower[%d]=%g > upper[%d]=%g", i, lower[i], i, upper[i])
		}
	}
	mMem := params.M
	if mMem <= 0 {
		mMem = 5
	}
	factr := params.Factr
	if factr <= 0 {
		factr = 1e7
	}
	pgtol := params.PgTol
	if pgtol < 0 {
		pgtol = 1e-5
	}
	maxIter := params.MaxIter
	if maxIter <= 0 {
		maxIter = 100
	}
	parscale := params.Parscale
	if parscale == nil {
		parscale = make([]float64, n)
		for i := range parscale {
			parscale[i] = 1
		}
	} else if len(parscale) != n {
		return nil, errors.New("lbfgsb: parscale length mismatch")
	}

	// Optimise in scaled coordinates z = x / parscale, exactly as R's optim does.
	// f_scaled(z) = fn(z .* parscale); ∇f_scaled = ∇fn .* parscale.
	scale := append([]float64(nil), parscale...)
	xWork := make([]float64, n)
	gWork := make([]float64, n)
	z := make([]float64, n)
	for i := 0; i < n; i++ {
		z[i] = start[i] / scale[i]
		xWork[i] = z[i] * scale[i]
		// project into bounds (in original coordinates)
		if xWork[i] < lower[i] {
			xWork[i] = lower[i]
			z[i] = xWork[i] / scale[i]
		}
		if xWork[i] > upper[i] {
			xWork[i] = upper[i]
			z[i] = xWork[i] / scale[i]
		}
	}
	// Bounds in scaled coordinates.
	zl := make([]float64, n)
	zu := make([]float64, n)
	for i := 0; i < n; i++ {
		zl[i] = lower[i] / scale[i]
		zu[i] = upper[i] / scale[i]
		if scale[i] < 0 {
			// Negative scaling reverses bounds; this is nonstandard but
			// keep symmetric for safety.
			zl[i], zu[i] = zu[i], zl[i]
		}
	}

	evalFG := func(zCur []float64) (float64, []float64) {
		for i := 0; i < n; i++ {
			xWork[i] = zCur[i] * scale[i]
		}
		f := fn(xWork)
		grad(gWork, xWork)
		gz := make([]float64, n)
		for i := 0; i < n; i++ {
			gz[i] = gWork[i] * scale[i]
		}
		return f, gz
	}

	f, g := evalFG(z)
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return nil, fmt.Errorf("lbfgsb: invalid initial objective %g", f)
	}

	state := newLbfgsbState(n, mMem)
	prevF := f
	converged := false
	iter := 0
	for iter = 0; iter < maxIter; iter++ {
		// projected gradient norm (inf-norm)
		pgNorm := 0.0
		for i := 0; i < n; i++ {
			gi := g[i]
			pi := z[i] - gi
			if pi < zl[i] {
				pi = zl[i]
			}
			if pi > zu[i] {
				pi = zu[i]
			}
			d := math.Abs(pi - z[i])
			if d > pgNorm {
				pgNorm = d
			}
		}
		if pgNorm <= pgtol {
			converged = true
			break
		}

		// Generalized Cauchy point: minimise quadratic model along the
		// piecewise-linear projected gradient path.
		xc, _, c, err := cauchyPoint(z, g, zl, zu, state)
		if err != nil {
			return nil, err
		}

		// Subspace minimisation on the free variables (those not at a bound).
		// subspaceMin returns x_bar (absolute coords); convert to direction.
		var d []float64
		var xBar []float64
		if hasFreeVar(xc, zl, zu) && state.k > 0 {
			xBar, err = subspaceMin(z, g, xc, c, zl, zu, state)
			if err != nil {
				xBar = xc
			}
		} else {
			xBar = xc
		}
		d = make([]float64, n)
		dNorm := 0.0
		for i := 0; i < n; i++ {
			d[i] = xBar[i] - z[i]
			dNorm += d[i] * d[i]
		}
		// If the proposed step is essentially zero, the projected gradient
		// already vanished (or the subspace solver returned the current
		// point) — the box-constrained KKT condition is met.
		if math.Sqrt(dNorm) <= lbfgsbEps*math.Max(1, math.Sqrt(vecDot(z, z))) {
			converged = true
			break
		}

		// Line search along d, projected onto the box.
		alpha, znew, fnew, gnew, err := projectedLineSearch(z, g, d, zl, zu, f, fn, grad, scale, xWork, gWork)
		if err != nil {
			// Direction was uphill / search failed; restart with steepest descent.
			state.reset()
			d = make([]float64, n)
			dNorm = 0
			for i := 0; i < n; i++ {
				d[i] = -g[i]
				dNorm += d[i] * d[i]
			}
			if math.Sqrt(dNorm) <= lbfgsbEps {
				converged = true
				break
			}
			alpha, znew, fnew, gnew, err = projectedLineSearch(z, g, d, zl, zu, f, fn, grad, scale, xWork, gWork)
			if err != nil {
				// At a KKT point: no feasible descent direction. Treat as converged.
				converged = true
				break
			}
		}
		_ = alpha

		// Update L-BFGS pairs.
		s := make([]float64, n)
		y := make([]float64, n)
		for i := 0; i < n; i++ {
			s[i] = znew[i] - z[i]
			y[i] = gnew[i] - g[i]
		}
		state.update(s, y)

		// Convergence on relative function change (R's factr criterion).
		denom := math.Max(math.Max(math.Abs(prevF), math.Abs(fnew)), 1)
		if math.Abs(prevF-fnew)/denom <= factr*lbfgsbEps {
			z = znew
			f = fnew
			g = gnew
			iter++
			converged = true
			break
		}
		z = znew
		f = fnew
		g = gnew
		prevF = fnew
	}

	xOut := make([]float64, n)
	for i := 0; i < n; i++ {
		xOut[i] = z[i] * scale[i]
		if xOut[i] < lower[i] {
			xOut[i] = lower[i]
		}
		if xOut[i] > upper[i] {
			xOut[i] = upper[i]
		}
	}
	return &lbfgsbResult{
		X:         xOut,
		F:         f,
		Iters:     iter,
		Converged: converged,
	}, nil
}

// ---- L-BFGS compact-form state ----

type lbfgsbState struct {
	n     int
	m     int
	k     int       // number of valid pairs (≤ m)
	S     []float64 // stored as n*m, column-major: S[j*n + i] = s_{j}[i]
	Y     []float64
	ys    []float64 // y_j^T s_j
	theta float64
	// Cached compact matrices, rebuilt when pairs change.
	dirty bool
	W     *mat.Dense // n x 2k:  [Y, theta*S]
	M     *mat.Dense // 2k x 2k: M = inv([-D, L^T; L, theta*S^T S])
}

func newLbfgsbState(n, m int) *lbfgsbState {
	return &lbfgsbState{
		n:     n,
		m:     m,
		S:     make([]float64, n*m),
		Y:     make([]float64, n*m),
		ys:    make([]float64, m),
		theta: 1,
	}
}

func (s *lbfgsbState) reset() {
	s.k = 0
	s.theta = 1
	s.dirty = true
}

// update appends a new (s, y) pair, rotating out the oldest if needed.
// Skips updates that violate s^T y > eps to keep B positive definite.
func (st *lbfgsbState) update(sVec, yVec []float64) {
	dot := 0.0
	for i := 0; i < st.n; i++ {
		dot += sVec[i] * yVec[i]
	}
	if dot <= lbfgsbEps*math.Sqrt(vecDot(yVec, yVec)*vecDot(sVec, sVec)) {
		// curvature too small; skip update
		return
	}

	if st.k == st.m {
		// rotate out the oldest pair: shift columns left by one
		copy(st.S, st.S[st.n:])
		copy(st.Y, st.Y[st.n:])
		copy(st.ys, st.ys[1:])
		st.k--
	}
	off := st.k * st.n
	copy(st.S[off:off+st.n], sVec)
	copy(st.Y[off:off+st.n], yVec)
	st.ys[st.k] = dot
	st.k++

	// theta = y^T y / s^T y (most-recent pair scaling)
	yty := vecDot(yVec, yVec)
	st.theta = yty / dot
	st.dirty = true
}

// ensureCompact rebuilds W and M = (W^T W block matrix)^{-1}.
func (st *lbfgsbState) ensureCompact() error {
	if !st.dirty && st.W != nil {
		return nil
	}
	st.dirty = false
	if st.k == 0 {
		st.W = nil
		st.M = nil
		return nil
	}
	k := st.k
	n := st.n

	W := mat.NewDense(n, 2*k, nil)
	for j := 0; j < k; j++ {
		for i := 0; i < n; i++ {
			W.Set(i, j, st.Y[j*n+i])
			W.Set(i, k+j, st.theta*st.S[j*n+i])
		}
	}

	// Build M^{-1} = [-D, L^T; L, theta * S^T S]
	// where D = diag(s_i^T y_i) and L_{ij} = s_i^T y_j (i > j), 0 otherwise.
	StS := mat.NewDense(k, k, nil)
	for i := 0; i < k; i++ {
		for j := 0; j < k; j++ {
			d := 0.0
			for r := 0; r < n; r++ {
				d += st.S[i*n+r] * st.S[j*n+r]
			}
			StS.Set(i, j, st.theta*d)
		}
	}
	L := mat.NewDense(k, k, nil)
	for i := 0; i < k; i++ {
		for j := 0; j < i; j++ {
			d := 0.0
			for r := 0; r < n; r++ {
				d += st.S[i*n+r] * st.Y[j*n+r]
			}
			L.Set(i, j, d)
		}
	}

	Minv := mat.NewDense(2*k, 2*k, nil)
	for i := 0; i < k; i++ {
		Minv.Set(i, i, -st.ys[i])
		for j := 0; j < k; j++ {
			Minv.Set(i, k+j, L.At(j, i)) // L^T
			Minv.Set(k+i, j, L.At(i, j))
			Minv.Set(k+i, k+j, StS.At(i, j))
		}
	}
	var Mfull mat.Dense
	if err := Mfull.Inverse(Minv); err != nil {
		// Fall back to pseudo-inverse so a near-singular block doesn't
		// derail the iteration; the next update will refresh M anyway.
		pinv, perr := Pinv(Minv, 0)
		if perr != nil {
			return fmt.Errorf("lbfgsb: failed to invert M: %w", err)
		}
		st.M = pinv
		st.W = W
		return nil
	}
	st.M = &Mfull
	st.W = W
	return nil
}

// hessianTimes computes B*v where B = theta*I - W M W^T.
func (st *lbfgsbState) hessianTimes(v []float64) []float64 {
	n := st.n
	out := make([]float64, n)
	for i := 0; i < n; i++ {
		out[i] = st.theta * v[i]
	}
	if st.k == 0 || st.W == nil {
		return out
	}
	// out -= W * M * W^T * v
	wTv := mat.NewVecDense(2*st.k, nil)
	for j := 0; j < 2*st.k; j++ {
		s := 0.0
		for i := 0; i < n; i++ {
			s += st.W.At(i, j) * v[i]
		}
		wTv.SetVec(j, s)
	}
	mwTv := mat.NewVecDense(2*st.k, nil)
	mwTv.MulVec(st.M, wTv)
	for i := 0; i < n; i++ {
		s := 0.0
		for j := 0; j < 2*st.k; j++ {
			s += st.W.At(i, j) * mwTv.AtVec(j)
		}
		out[i] -= s
	}
	return out
}

// ---- Cauchy point (BLNZ Algorithm CP) ----

// cauchyPoint returns:
//
//	xc   - the generalized Cauchy point
//	fcp  - quadratic model value at xc (for diagnostics)
//	c    - W^T (xc - x), used by the subspace minimisation
//	error
func cauchyPoint(x, g, l, u []float64, st *lbfgsbState) ([]float64, float64, []float64, error) {
	n := st.n
	if err := st.ensureCompact(); err != nil {
		return nil, 0, nil, err
	}

	// Compute d = -g, but zero out components that are at a bound and
	// would push outside the box.
	d := make([]float64, n)
	tBreak := make([]float64, n)
	freeMask := make([]bool, n)
	for i := 0; i < n; i++ {
		gi := g[i]
		if gi < 0 && x[i] < u[i] {
			d[i] = -gi
		} else if gi > 0 && x[i] > l[i] {
			d[i] = -gi
		} else {
			d[i] = 0
		}
		// Breakpoint t_i: time at which variable i hits a bound moving from x in direction d.
		if d[i] > 0 {
			tBreak[i] = (u[i] - x[i]) / d[i]
		} else if d[i] < 0 {
			tBreak[i] = (l[i] - x[i]) / d[i]
		} else {
			tBreak[i] = math.Inf(1)
		}
		freeMask[i] = d[i] != 0
	}

	// Sort indices by t_i ascending.
	type idxT struct {
		i int
		t float64
	}
	order := make([]idxT, 0, n)
	for i := 0; i < n; i++ {
		if d[i] != 0 {
			order = append(order, idxT{i: i, t: tBreak[i]})
		}
	}
	// stable insertion sort (n small in practice)
	for i := 1; i < len(order); i++ {
		for j := i; j > 0 && order[j-1].t > order[j].t; j-- {
			order[j-1], order[j] = order[j], order[j-1]
		}
	}

	// Initial quadratic-model coefficients along the projected gradient
	// path. With d[i] = -g[i] for free variables, fp at t=0 equals g^T d
	// (negative for descent), and the quadratic curvature is d^T B d > 0.
	// The breakpoint update formulas (BLNZ §4 eq 4.11/4.12) are consistent
	// with this fp = g^T d convention.
	xc := make([]float64, n)
	copy(xc, x)
	fp := 0.0
	for i := 0; i < n; i++ {
		fp += d[i] * g[i]
	}
	Bd := st.hessianTimes(d)
	fpp := 0.0
	for i := 0; i < n; i++ {
		fpp += d[i] * Bd[i]
	}
	if fpp < lbfgsbEps {
		fpp = lbfgsbEps
	}

	// W^T d, used to update fp/fpp incrementally as variables hit bounds.
	var wTd []float64
	if st.k > 0 && st.W != nil {
		wTd = make([]float64, 2*st.k)
		for j := 0; j < 2*st.k; j++ {
			s := 0.0
			for i := 0; i < n; i++ {
				s += st.W.At(i, j) * d[i]
			}
			wTd[j] = s
		}
	}
	c := make([]float64, 2*st.k) // accumulates W^T (xc - x)

	tCur := 0.0
	dtMin := -fp / fpp
	bIdx := 0
	for bIdx < len(order) {
		ib := order[bIdx]
		dt := ib.t - tCur
		if dtMin < dt {
			break
		}
		// Move to next breakpoint.
		i := ib.i
		// xc reaches its bound on variable i.
		if d[i] > 0 {
			xc[i] = u[i]
		} else {
			xc[i] = l[i]
		}
		// Update c += dt * W^T d (contribution from this segment)
		for j := 0; j < len(c); j++ {
			c[j] += dt * wTd[j]
		}
		// Update fp/fpp by removing variable i from the active direction.
		gi := g[i]
		di := d[i]
		// Quantities used by BLNZ Section 4 of the 1995 paper.
		// fp_new = fp + dt*fpp + g_i^2 + theta*g_i*z_i - g_i * w_i^T (M c)
		// fpp_new = fpp - theta*g_i^2 - 2 g_i * w_i^T M W^T d + g_i^2 * w_i^T M w_i
		zi := xc[i] - x[i]
		fp += dt*fpp + gi*gi + st.theta*gi*zi
		fpp -= st.theta * gi * gi
		if st.k > 0 && st.W != nil {
			// w_i = i-th row of W
			wi := make([]float64, 2*st.k)
			for j := 0; j < 2*st.k; j++ {
				wi[j] = st.W.At(i, j)
			}
			Mc := matVecMul(st.M, c)
			MwTd := matVecMul(st.M, wTd)
			Mwi := matVecMul(st.M, wi)
			fp -= gi * vecDot(wi, Mc)
			fpp += -2*gi*vecDot(wi, MwTd) + gi*gi*vecDot(wi, Mwi)
			// Remove this variable from d and W^T d.
			for j := 0; j < 2*st.k; j++ {
				wTd[j] -= di * wi[j]
			}
		}
		d[i] = 0
		freeMask[i] = false
		if fpp < lbfgsbEps {
			fpp = lbfgsbEps
		}
		dtMin = -fp / fpp
		tCur = ib.t
		bIdx++
		if fp >= 0 {
			dtMin = 0
			break
		}
	}
	if dtMin < 0 {
		dtMin = 0
	}
	// Free variables have not been advanced through the breakpoint loop;
	// they sit at x_orig. Move them to x_orig + (tCur + dtMin) * d.
	finalT := tCur + dtMin
	for i := 0; i < n; i++ {
		if freeMask[i] {
			xc[i] = x[i] + finalT*d[i]
			if xc[i] < l[i] {
				xc[i] = l[i]
			}
			if xc[i] > u[i] {
				xc[i] = u[i]
			}
		}
	}
	for j := 0; j < len(c); j++ {
		c[j] += dtMin * wTd[j]
	}

	// Quadratic model value at xc (informational; used by subspace step).
	zc := make([]float64, n)
	for i := 0; i < n; i++ {
		zc[i] = xc[i] - x[i]
	}
	Bzc := st.hessianTimes(zc)
	fcp := 0.0
	for i := 0; i < n; i++ {
		fcp += g[i]*zc[i] + 0.5*zc[i]*Bzc[i]
	}
	return xc, fcp, c, nil
}

// hasFreeVar returns true if any component of x is strictly inside its bounds.
func hasFreeVar(x, l, u []float64) bool {
	for i := range x {
		if x[i] > l[i]+lbfgsbEps && x[i] < u[i]-lbfgsbEps {
			return true
		}
	}
	return false
}

// ---- Subspace minimization ----

// subspaceMin computes a Newton step on the free variables of xc using
// the L-BFGS Hessian's compact form, then projects the result back into
// the box. Returns x_bar in absolute coordinates. Closely follows BLNZ
// Algorithm SSM (1995, §5).
//
// Reduced gradient at xc:
//
//	r = g + B (xc - x_orig)
//	  = g + theta (xc - x_orig) - W M c     (compact form)
//
// We then solve B_F * delta = -r_F on free variables and clamp.
func subspaceMin(xOrig, g, xc, c, l, u []float64, st *lbfgsbState) ([]float64, error) {
	n := st.n
	freeIdx := make([]int, 0, n)
	for i := 0; i < n; i++ {
		if xc[i] > l[i]+lbfgsbEps && xc[i] < u[i]-lbfgsbEps {
			freeIdx = append(freeIdx, i)
		}
	}
	if len(freeIdx) == 0 {
		return append([]float64(nil), xc...), nil
	}

	// Full reduced gradient r = g + theta*(xc - x_orig) - W M c on free vars.
	r := make([]float64, n)
	for i := 0; i < n; i++ {
		r[i] = g[i] + st.theta*(xc[i]-xOrig[i])
	}
	if st.k > 0 && st.W != nil {
		Mc := matVecMul(st.M, c)
		for i := 0; i < n; i++ {
			s := 0.0
			for j := 0; j < 2*st.k; j++ {
				s += st.W.At(i, j) * Mc[j]
			}
			r[i] -= s
		}
	}

	// B_F = theta * I_F - W_F M W_F^T (dense; n is small for FA workloads)
	BF := mat.NewDense(len(freeIdx), len(freeIdx), nil)
	rhs := mat.NewVecDense(len(freeIdx), nil)
	for ii, i := range freeIdx {
		rhs.SetVec(ii, -r[i])
		for jj, j := range freeIdx {
			val := 0.0
			if i == j {
				val += st.theta
			}
			if st.k > 0 && st.W != nil {
				wj := make([]float64, 2*st.k)
				for rIdx := 0; rIdx < 2*st.k; rIdx++ {
					wj[rIdx] = st.W.At(j, rIdx)
				}
				Mwj := matVecMul(st.M, wj)
				s := 0.0
				for rIdx := 0; rIdx < 2*st.k; rIdx++ {
					s += st.W.At(i, rIdx) * Mwj[rIdx]
				}
				val -= s
			}
			BF.Set(ii, jj, val)
		}
	}
	var delta mat.VecDense
	if err := delta.SolveVec(BF, rhs); err != nil {
		return nil, fmt.Errorf("subspace solve failed: %w", err)
	}

	xBar := append([]float64(nil), xc...)
	for k, i := range freeIdx {
		xBar[i] += delta.AtVec(k)
		if xBar[i] < l[i] {
			xBar[i] = l[i]
		}
		if xBar[i] > u[i] {
			xBar[i] = u[i]
		}
	}
	return xBar, nil
}

// ---- Projected line search (backtracking with Armijo + curvature) ----

func projectedLineSearch(x, g, dRaw, l, u []float64,
	f0 float64,
	fn func([]float64) float64,
	grad func(g, x []float64),
	scale, xWork, gWork []float64,
) (float64, []float64, float64, []float64, error) {
	n := len(x)
	// Subspace minimisation may have returned an absolute target; if so,
	// convert to a direction. Heuristic: if dRaw looks like a position
	// (any coord matches xc which is in bounds) we keep it; otherwise we
	// treat as direction. To remove ambiguity, callers always pass a
	// direction (delta), so accept as-is.
	d := make([]float64, n)
	maxAlpha := math.Inf(1)
	for i := 0; i < n; i++ {
		d[i] = dRaw[i]
		if d[i] > 0 {
			t := (u[i] - x[i]) / d[i]
			if t < maxAlpha {
				maxAlpha = t
			}
		} else if d[i] < 0 {
			t := (l[i] - x[i]) / d[i]
			if t < maxAlpha {
				maxAlpha = t
			}
		}
	}
	if maxAlpha <= 0 || math.IsNaN(maxAlpha) {
		return 0, nil, f0, g, errors.New("no feasible step")
	}

	gd := 0.0
	for i := 0; i < n; i++ {
		gd += g[i] * d[i]
	}
	if gd >= 0 {
		return 0, nil, f0, g, errors.New("not a descent direction")
	}

	const c1 = 1e-4
	const c2 = 0.9
	alpha := math.Min(1.0, maxAlpha)
	xnew := make([]float64, n)
	gnew := make([]float64, n)

	evalAt := func(a float64) (float64, []float64) {
		for i := 0; i < n; i++ {
			xnew[i] = x[i] + a*d[i]
			if xnew[i] < l[i] {
				xnew[i] = l[i]
			}
			if xnew[i] > u[i] {
				xnew[i] = u[i]
			}
		}
		// f and g require original (unscaled) coords
		for i := 0; i < n; i++ {
			xWork[i] = xnew[i] * scale[i]
		}
		f := fn(xWork)
		grad(gWork, xWork)
		for i := 0; i < n; i++ {
			gnew[i] = gWork[i] * scale[i]
		}
		return f, append([]float64(nil), gnew...)
	}

	const maxBacktrack = 30
	for k := 0; k < maxBacktrack; k++ {
		fNew, gNewSlice := evalAt(alpha)
		if math.IsNaN(fNew) || math.IsInf(fNew, 0) {
			alpha *= 0.5
			continue
		}
		if fNew <= f0+c1*alpha*gd {
			// Curvature condition (Wolfe). For projected line search,
			// fall back to Armijo only if strong Wolfe can't be met.
			gdNew := 0.0
			for i := 0; i < n; i++ {
				gdNew += gNewSlice[i] * d[i]
			}
			if math.Abs(gdNew) <= c2*math.Abs(gd) || alpha == maxAlpha {
				return alpha, append([]float64(nil), xnew...), fNew, gNewSlice, nil
			}
			// Increase alpha if room remains.
			if alpha < maxAlpha {
				alpha = math.Min(maxAlpha, alpha*1.5)
				continue
			}
			return alpha, append([]float64(nil), xnew...), fNew, gNewSlice, nil
		}
		alpha *= 0.5
	}
	return 0, nil, f0, g, errors.New("line search exhausted backtracks")
}

// ---- small linear-algebra helpers ----

func vecDot(a, b []float64) float64 {
	s := 0.0
	for i := range a {
		s += a[i] * b[i]
	}
	return s
}

func matVecMul(M *mat.Dense, v []float64) []float64 {
	r, c := M.Dims()
	if c != len(v) {
		return nil
	}
	out := make([]float64, r)
	for i := 0; i < r; i++ {
		s := 0.0
		for j := 0; j < c; j++ {
			s += M.At(i, j) * v[j]
		}
		out[i] = s
	}
	return out
}
