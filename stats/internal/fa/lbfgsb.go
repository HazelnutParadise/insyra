// fa/lbfgsb.go
//
// Pure-Go port of the L-BFGS-B v3.0 reference implementation by Ciyou Zhu,
// Richard Byrd, Jorge Nocedal and Jose Luis Morales (3-clause BSD). The
// goal is bit-near-perfect parity with R's optim(method = "L-BFGS-B"),
// which wraps the same Fortran. The port is split across several files:
//
//   lbfgsb.go         - public entry, lbfgsbState, public driver loop
//   lbfgsb_blas.go    - BLAS-1 (daxpy/dcopy/ddot/dnrm2/dscal) + dpofa/dtrsl
//   lbfgsb_lnsrch.go  - dcsrch + dcstep (Moré-Thuente line search)
//   lbfgsb_setulb.go  - setulb / mainlb driver, errclb, active, projgr,
//                       cmprlb, freev, hpsolb, matupd
//   lbfgsb_cauchy.go  - cauchy (generalized Cauchy point) + bmv
//   lbfgsb_form.go    - formk, formt
//   lbfgsb_subsm.go   - subsm (subspace minimization)
//
// All matrices are stored Fortran-style (column-major flattened []float64)
// so BLAS calls and stride/`incx` arguments translate verbatim.
package fa

import (
	"errors"
	"fmt"
	"math"
)

// lbfgsbResult is the outcome of an L-BFGS-B run.
type lbfgsbResult struct {
	X         []float64
	F         float64
	Iters     int
	Converged bool
	Task      string
}

// lbfgsbParams configures the optimizer. Defaults mirror R's optim().
type lbfgsbParams struct {
	M        int       // memory length (R default 5)
	Factr    float64   // relative-f tolerance multiplier of eps (R default 1e7)
	PgTol    float64   // projected-gradient inf-norm tolerance (R default 0)
	MaxIter  int       // iteration cap
	Parscale []float64 // parameter scales (R default rep(1, n))
	// Trace, if non-nil, is invoked after every fn evaluation in original
	// (un-scaled) coordinates: cb(evalIdx, f, x, g). Diagnostic only.
	Trace func(evalIdx int, f float64, x, g []float64)
}

// lbfgsb minimizes fn subject to lower <= x <= upper using the L-BFGS-B
// algorithm. grad must populate g with the gradient at the current x.
//
// Bounds: lower[i] = -Inf and upper[i] = +Inf indicate "unbounded on that
// side". Internally the BLNZ nbd[i] code is set as follows:
//   - 0 : no bounds
//   - 1 : lower bound only
//   - 2 : both lower and upper
//   - 3 : upper bound only
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
			return nil, fmt.Errorf("lbfgsb: lower[%d]=%g > upper[%d]=%g",
				i, lower[i], i, upper[i])
		}
	}

	// --- Defaults ---
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
		pgtol = 0
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

	// --- Bound code (nbd) and scaled bounds, mirroring R's optim() ---
	scale := append([]float64(nil), parscale...)
	nbd := make([]int, n)
	zl := make([]float64, n)
	zu := make([]float64, n)
	for i := 0; i < n; i++ {
		hasL := !math.IsInf(lower[i], -1)
		hasU := !math.IsInf(upper[i], 1)
		switch {
		case hasL && hasU:
			nbd[i] = 2
		case hasL && !hasU:
			nbd[i] = 1
		case !hasL && hasU:
			nbd[i] = 3
		default:
			nbd[i] = 0
		}
		if hasL {
			zl[i] = lower[i] / scale[i]
		}
		if hasU {
			zu[i] = upper[i] / scale[i]
		}
		if scale[i] < 0 {
			zl[i], zu[i] = zu[i], zl[i]
		}
	}

	// Scaled state: optimise in z = x / parscale, exactly as R's optim does.
	z := make([]float64, n)
	for i := 0; i < n; i++ {
		z[i] = start[i] / scale[i]
	}

	xWork := make([]float64, n)
	gWork := make([]float64, n)
	evalIdx := 0
	evalFG := func(zCur []float64) (float64, []float64) {
		for i := 0; i < n; i++ {
			xWork[i] = zCur[i] * scale[i]
		}
		f := fn(xWork)
		grad(gWork, xWork)
		evalIdx++
		if params.Trace != nil {
			xCopy := append([]float64(nil), xWork...)
			gCopy := append([]float64(nil), gWork...)
			params.Trace(evalIdx, f, xCopy, gCopy)
		}
		gz := make([]float64, n)
		for i := 0; i < n; i++ {
			gz[i] = gWork[i] * scale[i]
		}
		return f, gz
	}

	// Run the BLNZ driver. setulb returns task/state; we re-evaluate fn/grad
	// when task == "FG".
	res, err := setulbDriver(n, mMem, z, zl, zu, nbd, factr, pgtol, maxIter, evalFG)
	if err != nil {
		return nil, err
	}

	// Unscale back to original coordinates and project (defensive).
	xOut := make([]float64, n)
	for i := 0; i < n; i++ {
		xOut[i] = res.X[i] * scale[i]
		if !math.IsInf(lower[i], -1) && xOut[i] < lower[i] {
			xOut[i] = lower[i]
		}
		if !math.IsInf(upper[i], 1) && xOut[i] > upper[i] {
			xOut[i] = upper[i]
		}
	}
	return &lbfgsbResult{
		X:         xOut,
		F:         res.F,
		Iters:     res.Iters,
		Converged: res.Converged,
		Task:      res.Task,
	}, nil
}
