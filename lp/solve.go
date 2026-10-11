package lp

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/lpgen"
)

// Engine selects the solver that runs a model.
type Engine string

const (
	// EngineMILP is go-milp, a pure-Go solver. It needs nothing installed, and
	// the zero Options selects it.
	EngineMILP Engine = "milp"
	// EngineGLPK runs an installed glpsol, found through GLPK_PATH (the
	// executable or the directory holding it) and then PATH.
	EngineGLPK Engine = "glpk"
)

// Options configures one solve. The zero value solves with go-milp and no time
// limit.
type Options struct {
	Engine Engine
	// TimeLimit stops the search once it has run this long; zero means no
	// limit. GLPK takes whole seconds, so its limit is rounded up.
	TimeLimit time.Duration
}

// Status is what a solve found. A model without a solution is an answer, so
// every Status comes with a nil error.
type Status int

const (
	// StatusOptimal is a proven optimum.
	StatusOptimal Status = iota + 1
	// StatusFeasible means the time limit stopped the search holding a
	// solution that was not proven optimal.
	StatusFeasible
	// StatusInfeasible means the model has no solution.
	StatusInfeasible
	// StatusUnbounded means the objective can improve without limit.
	StatusUnbounded
	// StatusStopped means the time limit stopped the search before any
	// solution was found.
	StatusStopped
)

func (s Status) String() string {
	switch s {
	case StatusOptimal:
		return "optimal"
	case StatusFeasible:
		return "feasible"
	case StatusInfeasible:
		return "infeasible"
	case StatusUnbounded:
		return "unbounded"
	case StatusStopped:
		return "stopped"
	}
	return fmt.Sprintf("Status(%d)", int(s))
}

// Solution is the outcome of a solve. Objective and Values are set only when
// Status is StatusOptimal or StatusFeasible.
type Solution struct {
	Status    Status
	Objective float64
	// Values maps each variable's name to its value. It is nil when there is
	// no solution.
	Values  map[string]float64
	Engine  Engine
	Elapsed time.Duration
	// Nodes is the number of branch-and-bound nodes go-milp explored. GLPK
	// does not report it, so it is 0 for EngineGLPK.
	Nodes int
	// Log is what the engine reported. For EngineGLPK it is glpsol's output
	// followed by its report; for EngineMILP it is the warnings GLPK would
	// print while reading the same model, such as a missing End.
	Log string

	order []string // variable names in model order, for ToDataTable
}

var (
	// ErrInvalidModel means the model or LP file cannot be solved as written.
	// The wrapped error says why; for an LP file it names the line.
	ErrInvalidModel = errors.New("lp: the model cannot be solved as written")
	// ErrEngineUnavailable means Options.Engine names no known engine, or the
	// engine it names is not installed.
	ErrEngineUnavailable = errors.New("lp: the solver engine is not available")
	// ErrSolverFailed means the solver ran but gave no usable answer.
	ErrSolverFailed = errors.New("lp: the solver failed")
)

// Solve solves an lpgen model. An error means there is no Solution; an
// infeasible, unbounded or time-limited model is a Solution with a nil error.
func Solve(model *lpgen.LPModel, opts Options) (*Solution, error) {
	if model == nil {
		return nil, fmt.Errorf("%w: no model", ErrInvalidModel)
	}
	var text bytes.Buffer
	if err := model.WriteLP(&text); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidModel, err)
	}
	return solveText("model", text.Bytes(), opts)
}

// SolveFile solves a CPLEX LP file. An error means there is no Solution; an
// infeasible, unbounded or time-limited model is a Solution with a nil error.
// A file that cannot be read returns the underlying error, so
// errors.Is(err, fs.ErrNotExist) works for a missing file.
func SolveFile(path string, opts Options) (*Solution, error) {
	text, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("lp: reading %s: %w", path, err)
	}
	return solveText(path, text, opts)
}

// solveText runs one engine on LP text. name is what error messages call the
// text: the file path, or "model" for an lpgen model.
func solveText(name string, text []byte, opts Options) (*Solution, error) {
	start := time.Now()
	var (
		sol *Solution
		err error
	)
	switch opts.Engine {
	case "", EngineMILP:
		var p *problem
		if p, err = readLP(name, text); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrInvalidModel, err)
		}
		sol, err = solveMILP(p, opts.TimeLimit)
		if sol != nil {
			sol.Engine = EngineMILP
		}
	case EngineGLPK:
		sol, err = solveGLPK(name, text, opts.TimeLimit)
		if sol != nil {
			sol.Engine = EngineGLPK
		}
	default:
		return nil, fmt.Errorf("%w: unknown engine %q", ErrEngineUnavailable, opts.Engine)
	}
	if err != nil {
		return nil, err
	}
	sol.Elapsed = time.Since(start)
	return sol, nil
}

// ToDataTable returns a Variable column and a Value column in model order. It
// never returns nil; a Solution without values gives a table with no rows and
// no error, because no solution is an answer rather than a failure.
func (s *Solution) ToDataTable() *insyra.DataTable {
	names := insyra.NewDataList().SetName("Variable")
	values := insyra.NewDataList().SetName("Value")
	if s != nil && s.Values != nil {
		for _, name := range s.order {
			names.Append(name)
			values.Append(s.Values[name])
		}
	}
	return insyra.NewDataTable(names, values)
}
