// fa/lbfgsb_driver.go
//
// Port of subroutines setulb + mainlb from lbfgsb.f v3.0. Because Go has
// closures we collapse the Fortran reverse-communication driver into a
// single direct call: when mainlb wanted f / g it would set
// task = "FG_*" and return; here we evaluate directly via the caller's
// `eval` function.
package fa

import (
	"fmt"
	"math"
)

// driverResult is the internal rendezvous between the BLNZ driver loop and
// the public lbfgsb wrapper.
type driverResult struct {
	X         []float64
	F         float64
	Iters     int
	Converged bool
	Task      string
}

// setulbDriver runs L-BFGS-B from the initial point x to a local
// minimizer of f s.t. lower ≤ x ≤ upper. eval(x) must return f(x) and a
// freshly-allocated gradient slice ∇f(x).
//
// The implementation mirrors mainlb in lbfgsb.f line for line; the only
// structural change is that "FG_*" reverse-communication tasks are
// resolved by directly invoking eval inside this function. The matrix
// layouts (column-major) and 1-based indexing semantics are preserved by
// the subroutines this driver calls.
func setulbDriver(n, m int, x, l, u []float64, nbd []int,
	factr, pgtol float64, maxIter int,
	eval func(z []float64) (float64, []float64),
) (*driverResult, error) {
	// --- Validate inputs (errclb) ---
	if task, _, k := errclb(n, m, factr, l, u, nbd); task != "" {
		return nil, fmt.Errorf("lbfgsb: %s (k=%d)", task, k)
	}

	// --- Allocate work arrays following setulb's wa partitioning ---
	ws := make([]float64, n*m)
	wy := make([]float64, n*m)
	sy := make([]float64, m*m)
	ss := make([]float64, m*m)
	wt := make([]float64, m*m)
	wn := make([]float64, 4*m*m)
	snd := make([]float64, 4*m*m)
	z := make([]float64, n)
	r := make([]float64, n)
	d := make([]float64, n)
	tArr := make([]float64, n)
	xp := make([]float64, n)
	wa := make([]float64, 8*m)

	indexArr := make([]int, n)
	iwhere := make([]int, n)
	indx2 := make([]int, n)

	// --- Initial active-set classification + projection ---
	prjctd, cnstnd, boxed := active(n, l, u, nbd, x, iwhere)
	_ = prjctd

	// --- Initial f, g ---
	f, g := eval(x)
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return nil, fmt.Errorf("lbfgsb: invalid initial f = %g", f)
	}
	nfgv := 1

	// --- L-BFGS bookkeeping ---
	col := 0
	head := 1
	itail := 0
	iupdat := 0
	iter := 0
	nfree := n
	ifun := 0
	iback := 0
	updatd := false
	theta := 1.0
	fold := 0.0
	gd := 0.0
	gdold := 0.0
	stp := 0.0
	stpmx := 0.0
	dnorm := 0.0
	dtd := 0.0
	xstep := 0.0
	_ = xstep
	epsmch := 2.220446049250313e-16
	tol := factr * epsmch

	lnState := &lnsrlbState{}
	task := ""
	nintol := 0
	nskip := 0
	_ = nintol
	_ = nskip

	// --- Convergence check on the projected gradient ---
	sbgnrm := projgr(n, l, u, nbd, x, g)
	if sbgnrm <= pgtol {
		return &driverResult{
			X:         append([]float64(nil), x...),
			F:         f,
			Iters:     0,
			Converged: true,
			Task:      "CONVERGENCE: NORM_OF_PROJECTED_GRADIENT_<=_PGTOL",
		}, nil
	}

	// ============================ main loop ===============================
	for iter < maxIter {
		// (1) Generalized Cauchy point.
		var info int
		var wrk bool
		var ileave, nenter int

		if !cnstnd && col > 0 {
			dcopy(n, x, 1, z, 1)
			wrk = updatd
		} else {
			info = 0
			_, info = cauchy(n, x, l, u, nbd, g,
				indx2, iwhere, tArr, d, z, m, wy, ws, sy, wt,
				theta, col, head,
				wa[0:], wa[2*m:], wa[4*m:], wa[6*m:],
				sbgnrm, epsmch)
			if info != 0 {
				// Singular triangular system; refresh memory and retry.
				col, head, theta, iupdat, updatd = 0, 1, 1.0, 0, false
				continue
			}
			// freev: count entering / leaving free vars and partition index.
			wrk = freev(n, &nfree, indexArr, &nenter, &ileave, indx2,
				iwhere, updatd, cnstnd, iter)
		}
		nact := n - nfree
		_ = nact

		// (2) Subspace minimization (skip if no free vars or B = θI).
		if nfree != 0 && col != 0 {
			if wrk {
				if info = formk(n, nfree, indexArr, nenter, ileave, indx2,
					iupdat, updatd, wn, snd, m, ws, wy, sy, theta, col, head); info != 0 {
					// Cholesky failed; refresh.
					col, head, theta, iupdat, updatd = 0, 1, 1.0, 0, false
					continue
				}
			}
			info = cmprlb(n, m, x, g, ws, wy, sy, wt, z, r, wa,
				indexArr, theta, col, head, nfree, cnstnd)
			if info != 0 {
				// Singular bmv solve in cmprlb — refresh and restart.
				col, head, theta, iupdat, updatd = 0, 1, 1.0, 0, false
				continue
			}
			if _, info = subsm(n, m, nfree, indexArr, l, u, nbd, z, r, xp, ws, wy,
				theta, x, g, col, head, wa, wn); info != 0 {
				col, head, theta, iupdat, updatd = 0, 1, 1.0, 0, false
				continue
			}
		}

		// (3) Line search.
		for i := 0; i < n; i++ {
			d[i] = z[i] - x[i]
		}
		task = ""
		// reset counters that lnsrlb uses cumulatively
		ifun, iback = 0, 0
		// iterate the dcsrch reverse-communication loop directly
		lsTask := lnsrlb(n, l, u, nbd, x, &f, &fold, &gd, &gdold, g, d, r, tArr, z,
			&stp, &dnorm, &dtd, &xstep, &stpmx,
			iter, &ifun, &iback, &nfgv, &info,
			task, boxed, cnstnd, lnState)
		// Subsequent re-entries: lnsrlb expects task[:5] = "FG_LN".
		restart := false
		for {
			if info != 0 || iback >= 20 {
				// Restore previous iterate; refresh memory or abort.
				dcopy(n, tArr, 1, x, 1)
				dcopy(n, r, 1, g, 1)
				f = fold
				if col == 0 {
					return &driverResult{
						X:         append([]float64(nil), x...),
						F:         f,
						Iters:     iter + 1,
						Converged: false,
						Task:      "ABNORMAL_TERMINATION_IN_LNSRCH",
					}, nil
				}
				col, head, theta, iupdat, updatd = 0, 1, 1.0, 0, false
				restart = true
				break
			}
			if lsTask == "FG_LNSRCH" {
				fNew, gNew := eval(x)
				f = fNew
				copy(g, gNew)
				lsTask = lnsrlb(n, l, u, nbd, x, &f, &fold, &gd, &gdold,
					g, d, r, tArr, z,
					&stp, &dnorm, &dtd, &xstep, &stpmx,
					iter, &ifun, &iback, &nfgv, &info,
					"FG_LN", boxed, cnstnd, lnState)
				continue
			}
			// "NEW_X" — line search succeeded.
			break
		}
		if restart {
			continue
		}

		iter++

		// (4) Convergence checks after the new iterate.
		sbgnrm = projgr(n, l, u, nbd, x, g)
		if sbgnrm <= pgtol {
			return &driverResult{
				X:         append([]float64(nil), x...),
				F:         f,
				Iters:     iter,
				Converged: true,
				Task:      "CONVERGENCE: NORM_OF_PROJECTED_GRADIENT_<=_PGTOL",
			}, nil
		}
		ddum := math.Max(math.Max(math.Abs(fold), math.Abs(f)), 1)
		if (fold - f) <= tol*ddum {
			return &driverResult{
				X:         append([]float64(nil), x...),
				F:         f,
				Iters:     iter,
				Converged: true,
				Task:      "CONVERGENCE: REL_REDUCTION_OF_F_<=_FACTR*EPSMCH",
			}, nil
		}

		// (5) Curvature update r = g − r_old, dr = y'·s, rr = y'·y.
		for i := 0; i < n; i++ {
			r[i] = g[i] - r[i]
		}
		rr := ddot(n, r, 1, r, 1)
		var dr, ddumGS float64
		if stp == 1 {
			dr = gd - gdold
			ddumGS = -gdold
		} else {
			dr = (gd - gdold) * stp
			dscal(n, stp, d, 1)
			ddumGS = -gdold * stp
		}
		if dr <= epsmch*ddumGS {
			updatd = false
			nskip++
			continue
		}

		// (6) Update WS, WY, SY, SS, then form / factor T.
		updatd = true
		iupdat++
		matupd(n, m, ws, wy, sy, ss, d, r,
			&itail, &iupdat, &col, &head,
			&theta, rr, dr, stp, dtd)
		if info = formt(m, wt, sy, ss, col, theta); info != 0 {
			col, head, theta, iupdat, updatd = 0, 1, 1.0, 0, false
			continue
		}
	}

	return &driverResult{
		X:         append([]float64(nil), x...),
		F:         f,
		Iters:     iter,
		Converged: false,
		Task:      "MAX_ITERATIONS_REACHED",
	}, nil
}
