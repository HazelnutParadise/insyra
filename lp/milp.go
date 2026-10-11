package lp

import (
	"errors"
	"fmt"
	"math"
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
		rel := milp.LessEq
		switch r.rel {
		case greaterEq:
			rel = milp.GreaterEq
		case equalTo:
			rel = milp.EqualTo
		}
		m.AddConstraint(terms(r.terms), rel, r.rhs)
	}

	opts := milp.DefaultOptions()
	opts.TimeLimit = timeLimit
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
