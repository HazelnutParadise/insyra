// fa/lbfgsb_lnsrch.go
//
// Moré-Thuente line search (Minpack-2 dcsrch / dcstep) used by L-BFGS-B.
// Faithful line-by-line port of the Fortran reference.
//
// Convention: the algorithm is reverse-communication. The caller invokes
// dcsrch repeatedly; each call sets `task` to one of:
//   - "FG":   evaluate f and g at the current stp and call again
//   - "CONV": converged (Armijo + curvature both satisfied)
//   - "WARN": convergence not achievable (returns best step found)
//   - "ERROR: <msg>": invalid input
package fa

import "math"

// dcsrchState mirrors the Fortran isave[2] + dsave[13] persistent block.
type dcsrchState struct {
	brackt bool
	stage  int
	ginit  float64
	gtest  float64
	gx     float64
	gy     float64
	finit  float64
	fx     float64
	fy     float64
	stx    float64
	sty    float64
	stmin  float64
	stmax  float64
	width  float64
	width1 float64
}

// dcsrch finds a step that satisfies a sufficient decrease condition and
// a curvature condition. Direct port of MINPACK-2 dcsrch (lbfgsb.f).
//
// Inputs/outputs (matching Fortran):
//   - f, g, stp: objective value, derivative, and current step.
//   - ftol, gtol, xtol: tolerances for sufficient decrease, curvature,
//     and an acceptable interval width.
//   - stpmin, stpmax: positive bounds on the step.
//   - task: "START" on first call; "FG" while iterating; "CONV"/"WARN"/
//     "ERROR..." on exit.
//   - st: persistent state across calls (caller-owned).
func dcsrch(f, g *float64, stp *float64, ftol, gtol, xtol, stpmin, stpmax float64,
	task *string, st *dcsrchState,
) {
	const (
		p5     = 0.5
		p66    = 0.66
		xtrapl = 1.1
		xtrapu = 4.0
	)

	if len(*task) >= 5 && (*task)[:5] == "START" {
		// --- Validate inputs ---
		switch {
		case *stp < stpmin:
			*task = "ERROR: STP .LT. STPMIN"
		case *stp > stpmax:
			*task = "ERROR: STP .GT. STPMAX"
		case *g >= 0:
			*task = "ERROR: INITIAL G .GE. ZERO"
		case ftol < 0:
			*task = "ERROR: FTOL .LT. ZERO"
		case gtol < 0:
			*task = "ERROR: GTOL .LT. ZERO"
		case xtol < 0:
			*task = "ERROR: XTOL .LT. ZERO"
		case stpmin < 0:
			*task = "ERROR: STPMIN .LT. ZERO"
		case stpmax < stpmin:
			*task = "ERROR: STPMAX .LT. STPMIN"
		}
		if len(*task) >= 5 && (*task)[:5] == "ERROR" {
			return
		}
		// --- Initialize ---
		st.brackt = false
		st.stage = 1
		st.finit = *f
		st.ginit = *g
		st.gtest = ftol * st.ginit
		st.width = stpmax - stpmin
		st.width1 = st.width / p5
		st.stx = 0
		st.fx = st.finit
		st.gx = st.ginit
		st.sty = 0
		st.fy = st.finit
		st.gy = st.ginit
		st.stmin = 0
		st.stmax = *stp + xtrapu*(*stp)
		*task = "FG"
		return
	}
	// Continuing call: state already in st.

	ftest := st.finit + (*stp)*st.gtest

	if st.stage == 1 && *f <= ftest && *g >= 0 {
		st.stage = 2
	}

	// --- Warnings ---
	if st.brackt && (*stp <= st.stmin || *stp >= st.stmax) {
		*task = "WARNING: ROUNDING ERRORS PREVENT PROGRESS"
	}
	if st.brackt && st.stmax-st.stmin <= xtol*st.stmax {
		*task = "WARNING: XTOL TEST SATISFIED"
	}
	if *stp == stpmax && *f <= ftest && *g <= st.gtest {
		*task = "WARNING: STP = STPMAX"
	}
	if *stp == stpmin && (*f > ftest || *g >= st.gtest) {
		*task = "WARNING: STP = STPMIN"
	}

	// --- Test convergence ---
	if *f <= ftest && math.Abs(*g) <= gtol*(-st.ginit) {
		*task = "CONVERGENCE"
	}

	if len(*task) >= 4 && ((*task)[:4] == "WARN" || (*task)[:4] == "CONV") {
		return
	}

	// --- Compute next trial step via dcstep ---
	if st.stage == 1 && *f <= st.fx && *f > ftest {
		// Use the modified function psi(stp) = f(stp) - gtest*stp.
		fm := *f - (*stp)*st.gtest
		fxm := st.fx - st.stx*st.gtest
		fym := st.fy - st.sty*st.gtest
		gm := *g - st.gtest
		gxm := st.gx - st.gtest
		gym := st.gy - st.gtest

		dcstep(&st.stx, &fxm, &gxm, &st.sty, &fym, &gym, stp, &fm, &gm,
			&st.brackt, st.stmin, st.stmax)

		st.fx = fxm + st.stx*st.gtest
		st.fy = fym + st.sty*st.gtest
		st.gx = gxm + st.gtest
		st.gy = gym + st.gtest
	} else {
		dcstep(&st.stx, &st.fx, &st.gx, &st.sty, &st.fy, &st.gy, stp, f, g,
			&st.brackt, st.stmin, st.stmax)
	}

	// --- Bisection if width is not shrinking fast enough ---
	if st.brackt {
		if math.Abs(st.sty-st.stx) >= p66*st.width1 {
			*stp = st.stx + p5*(st.sty-st.stx)
		}
		st.width1 = st.width
		st.width = math.Abs(st.sty - st.stx)
	}

	// --- Update interval bounds for the next trial ---
	if st.brackt {
		st.stmin = math.Min(st.stx, st.sty)
		st.stmax = math.Max(st.stx, st.sty)
	} else {
		st.stmin = *stp + xtrapl*(*stp-st.stx)
		st.stmax = *stp + xtrapu*(*stp-st.stx)
	}

	// --- Project step back into [stpmin, stpmax] ---
	if *stp < stpmin {
		*stp = stpmin
	}
	if *stp > stpmax {
		*stp = stpmax
	}

	// --- Fall back to best known step if no further progress is possible ---
	if (st.brackt && (*stp <= st.stmin || *stp >= st.stmax)) ||
		(st.brackt && st.stmax-st.stmin <= xtol*st.stmax) {
		*stp = st.stx
	}

	// Request another function/gradient evaluation.
	*task = "FG"
}

// dcstep updates the bracketing interval [stx, sty] and computes the next
// trial step `stp` using a safeguarded interpolation. Direct port of
// MINPACK-2 dcstep.
func dcstep(stx, fx, dx *float64,
	sty, fy, dy *float64,
	stp *float64, fp, dp *float64,
	brackt *bool,
	stpmin, stpmax float64,
) {
	const (
		p66   = 0.66
		two   = 2.0
		three = 3.0
	)

	sgnd := (*dp) * ((*dx) / math.Abs(*dx))

	var stpf float64

	switch {
	// First case: a higher function value. Minimum is bracketed.
	case *fp > *fx:
		theta := three*(*fx-*fp)/(*stp-*stx) + *dx + *dp
		s := maxAbs3(theta, *dx, *dp)
		gamma := s * math.Sqrt((theta/s)*(theta/s)-(*dx/s)*(*dp/s))
		if *stp < *stx {
			gamma = -gamma
		}
		p := (gamma - *dx) + theta
		q := ((gamma - *dx) + gamma) + *dp
		r := p / q
		stpc := *stx + r*(*stp-*stx)
		stpq := *stx + ((*dx/((*fx-*fp)/(*stp-*stx)+*dx))/two)*(*stp-*stx)
		if math.Abs(stpc-*stx) < math.Abs(stpq-*stx) {
			stpf = stpc
		} else {
			stpf = stpc + (stpq-stpc)/two
		}
		*brackt = true

	// Second case: lower f, derivatives of opposite sign. Bracketed.
	case sgnd < 0:
		theta := three*(*fx-*fp)/(*stp-*stx) + *dx + *dp
		s := maxAbs3(theta, *dx, *dp)
		gamma := s * math.Sqrt((theta/s)*(theta/s)-(*dx/s)*(*dp/s))
		if *stp > *stx {
			gamma = -gamma
		}
		p := (gamma - *dp) + theta
		q := ((gamma - *dp) + gamma) + *dx
		r := p / q
		stpc := *stp + r*(*stx-*stp)
		stpq := *stp + (*dp/(*dp-*dx))*(*stx-*stp)
		if math.Abs(stpc-*stp) > math.Abs(stpq-*stp) {
			stpf = stpc
		} else {
			stpf = stpq
		}
		*brackt = true

	// Third case: lower f, derivatives same sign, |dp| < |dx|.
	case math.Abs(*dp) < math.Abs(*dx):
		theta := three*(*fx-*fp)/(*stp-*stx) + *dx + *dp
		s := maxAbs3(theta, *dx, *dp)
		gamma := s * math.Sqrt(math.Max(0, (theta/s)*(theta/s)-(*dx/s)*(*dp/s)))
		if *stp > *stx {
			gamma = -gamma
		}
		p := (gamma - *dp) + theta
		q := (gamma + (*dx - *dp)) + gamma
		r := p / q
		var stpc float64
		switch {
		case r < 0 && gamma != 0:
			stpc = *stp + r*(*stx-*stp)
		case *stp > *stx:
			stpc = stpmax
		default:
			stpc = stpmin
		}
		stpq := *stp + (*dp/(*dp-*dx))*(*stx-*stp)

		if *brackt {
			// minimizer bracketed: prefer cubic if closer to stp than secant.
			if math.Abs(stpc-*stp) < math.Abs(stpq-*stp) {
				stpf = stpc
			} else {
				stpf = stpq
			}
			if *stp > *stx {
				stpf = math.Min(*stp+p66*(*sty-*stp), stpf)
			} else {
				stpf = math.Max(*stp+p66*(*sty-*stp), stpf)
			}
		} else {
			// minimizer not yet bracketed: prefer cubic if farther from stp.
			if math.Abs(stpc-*stp) > math.Abs(stpq-*stp) {
				stpf = stpc
			} else {
				stpf = stpq
			}
			stpf = math.Min(stpmax, stpf)
			stpf = math.Max(stpmin, stpf)
		}

	// Fourth case: lower f, derivatives same sign, |dp| >= |dx|.
	default:
		if *brackt {
			theta := three*(*fp-*fy)/(*sty-*stp) + *dy + *dp
			s := maxAbs3(theta, *dy, *dp)
			gamma := s * math.Sqrt((theta/s)*(theta/s)-(*dy/s)*(*dp/s))
			if *stp > *sty {
				gamma = -gamma
			}
			p := (gamma - *dp) + theta
			q := ((gamma - *dp) + gamma) + *dy
			r := p / q
			stpc := *stp + r*(*sty-*stp)
			stpf = stpc
		} else if *stp > *stx {
			stpf = stpmax
		} else {
			stpf = stpmin
		}
	}

	// --- Update the interval that contains a minimizer ---
	if *fp > *fx {
		*sty = *stp
		*fy = *fp
		*dy = *dp
	} else {
		if sgnd < 0 {
			*sty = *stx
			*fy = *fx
			*dy = *dx
		}
		*stx = *stp
		*fx = *fp
		*dx = *dp
	}

	// New step
	*stp = stpf
}

func maxAbs3(a, b, c float64) float64 {
	m := math.Abs(a)
	if x := math.Abs(b); x > m {
		m = x
	}
	if x := math.Abs(c); x > m {
		m = x
	}
	return m
}
