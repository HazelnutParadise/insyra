package lp

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"
	"time"

	milp "github.com/daniel-sullivan/go-milp"
)

// solveMILP solves a problem with go-milp.
//
// go-milp returns the best solution it holds together with ErrTimeLimit when
// the time limit stops it. lp reports that as StatusFeasible with a nil error,
// so a caller who stops at `if err != nil` does not throw the solution away.
func solveMILP(p *problem, timeLimit time.Duration) (*Solution, error) {
	if err := checkBounds(p); err != nil {
		return nil, err
	}

	m := milp.NewModel()
	vars := make([]milp.Var, len(p.cols))
	for i, c := range p.cols {
		kind := milp.Continuous
		if c.integer {
			// Not milp.Binary: that forces [0, 1], and GLPK lets a Binary
			// variable keep a bound from the Bounds section.
			kind = milp.Integer
		}
		vars[i] = m.AddVar(c.lower, c.upper, kind)
	}
	terms := func(ts []term) []milp.Term {
		out := make([]milp.Term, len(ts))
		for i, t := range ts {
			out[i] = milp.Term{Var: vars[t.col], Coef: t.coef}
		}
		return out
	}
	sense := milp.Minimize
	if p.maximize {
		sense = milp.Maximize
	}
	m.SetObjective(sense, terms(p.objective))
	for _, r := range p.rows {
		// Normalize tiny rows before handing them to go-milp. Its LP solver
		// uses absolute feasibility tolerances, so unscaled coefficients below
		// that threshold can look like zero even with presolve disabled.
		scale := 1.0
		maxCoef := 0.0
		for _, t := range r.terms {
			if a := math.Abs(t.coef); a > maxCoef {
				maxCoef = a
			}
		}
		if maxCoef > 0 && maxCoef < 1 {
			scale = 1 / maxCoef
		}
		rowTerms := make([]term, len(r.terms))
		copy(rowTerms, r.terms)
		for i := range rowTerms {
			rowTerms[i].coef *= scale
		}
		rel := milp.LessEq
		switch r.rel {
		case greaterEq:
			rel = milp.GreaterEq
		case equalTo:
			rel = milp.EqualTo
		}
		m.AddConstraint(terms(rowTerms), rel, r.rhs*scale)
	}

	opts := milp.DefaultOptions()
	opts.TimeLimit = timeLimit
	opts.DisablePresolve = true
	res, stats, err := m.SolveWithStats(opts)

	limited := errors.Is(err, milp.ErrTimeLimit) || errors.Is(err, milp.ErrNodeLimit)
	sol := &Solution{Nodes: stats.Nodes, Log: strings.Join(p.warnings, "\n")}
	switch {
	case err != nil && !limited:
		if errors.Is(err, milp.ErrNumeric) {
			return nil, fmt.Errorf("%w: %w", ErrSolverFailed, err)
		}
		return nil, fmt.Errorf("%w: %w", ErrInvalidModel, err)
	case res == nil:
		return nil, fmt.Errorf("%w: go-milp returned no result", ErrSolverFailed)
	case limited && res.Values == nil:
		sol.Status = StatusStopped
		return sol, nil
	case limited:
		sol.Status = StatusFeasible
	case res.Status == milp.Optimal:
		sol.Status = StatusOptimal
	case res.Status == milp.Infeasible:
		sol.Status = StatusInfeasible
		return sol, nil
	case res.Status == milp.Unbounded:
		sol.Status = StatusUnbounded
		if hasIntegerVariables(p) {
			if hasImpossibleIntegerEquality(p) {
				sol.Status = StatusInfeasible
				return sol, nil
			}
			// An unbounded LP relaxation may have no integer-feasible point.
			// Solve the same constraints as a zero-objective feasibility problem
			// before reporting the relaxation's unbounded status.
			feasibilityProblem := *p
			feasibilityProblem.maximize = false
			feasibilityProblem.objective = nil
			// Branching on unbounded integer variables need not end, so the
			// check runs under the caller's limit or, without one, a cap of
			// its own; a check that cannot decide reports StatusStopped.
			checkLimit := timeLimit
			if checkLimit <= 0 {
				checkLimit = feasibilityCheckLimit
			}
			feasibility, err := solveMILP(&feasibilityProblem, checkLimit)
			if err != nil {
				return nil, err
			}
			switch feasibility.Status {
			case StatusInfeasible:
				sol.Status = StatusInfeasible
			case StatusOptimal, StatusFeasible:
				sol.Status = StatusUnbounded
			case StatusStopped:
				sol.Status = StatusStopped
			default:
				return nil, fmt.Errorf("%w: feasibility solve returned status %v", ErrSolverFailed, feasibility.Status)
			}
		}
		return sol, nil
	default:
		return nil, fmt.Errorf("%w: go-milp returned status %v", ErrSolverFailed, res.Status)
	}

	sol.Objective = res.Objective
	sol.Values = make(map[string]float64, len(p.cols))
	sol.order = make([]string, len(p.cols))
	for i, c := range p.cols {
		sol.Values[c.name] = res.Value(vars[i])
		sol.order[i] = c.name
	}
	return sol, nil
}

// feasibilityCheckLimit bounds the integer-feasibility check that follows an
// unbounded relaxation when the caller set no time limit.
const feasibilityCheckLimit = 30 * time.Second

func hasIntegerVariables(p *problem) bool {
	for _, col := range p.cols {
		if col.integer {
			return true
		}
	}
	return false
}

// hasImpossibleIntegerEquality proves infeasibility for an equality whose
// terms are all integer variables with integral coefficients when the gcd of
// those coefficients does not divide the right-hand side. This quick exact
// certificate also avoids an infinite branch ray for equations such as
// 2*x-2*y=1 before asking the MILP solver to check general feasibility.
func hasImpossibleIntegerEquality(p *problem) bool {
	for _, row := range p.rows {
		if row.rel != equalTo {
			continue
		}
		rhs := new(big.Rat).SetFloat64(row.rhs)
		if rhs == nil || !rhs.IsInt() {
			continue
		}
		gcd := new(big.Int)
		valid := true
		for _, item := range row.terms {
			if !p.cols[item.col].integer {
				valid = false
				break
			}
			coef := new(big.Rat).SetFloat64(item.coef)
			if coef == nil || !coef.IsInt() {
				valid = false
				break
			}
			abs := new(big.Int).Abs(new(big.Int).Set(coef.Num()))
			gcd.GCD(nil, nil, gcd, abs)
		}
		if !valid {
			continue
		}
		if gcd.Sign() == 0 {
			if rhs.Sign() != 0 {
				return true
			}
			continue
		}
		if new(big.Int).Rem(rhs.Num(), gcd).Sign() != 0 {
			return true
		}
	}
	return false
}

// checkBounds refuses the bounds GLPK refuses when it starts solving: a lower
// bound above the upper one, and a fractional bound on an integer variable.
// Reading accepts both, so without this check go-milp would call the second
// model infeasible where GLPK calls it invalid.
func checkBounds(p *problem) error {
	for _, c := range p.cols {
		if c.lower > c.upper {
			return fmt.Errorf("%w: variable %s has lower bound %g above upper bound %g", ErrInvalidModel, c.name, c.lower, c.upper)
		}
		if !c.integer {
			continue
		}
		for _, bound := range []float64{c.lower, c.upper} {
			if !math.IsInf(bound, 0) && bound != math.Trunc(bound) {
				return fmt.Errorf("%w: integer variable %s has non-integer bound %g", ErrInvalidModel, c.name, bound)
			}
		}
	}
	return nil
}
