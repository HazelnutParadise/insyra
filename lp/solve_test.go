package lp

import (
	"errors"
	"io/fs"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/lpgen"
)

// quietLP silences the warnings these tests provoke and restores the config.
func quietLP(t *testing.T) {
	t.Helper()
	level := insyra.Config.GetLogLevel()
	panicOnError := insyra.Config.GetPanicOnError()
	insyra.Config.SetLogLevel(insyra.LogLevelFatal)
	insyra.Config.SetPanicOnError(false)
	t.Cleanup(func() {
		insyra.Config.SetLogLevel(level)
		insyra.Config.SetPanicOnError(panicOnError)
	})
}

func fixtureLP(name string) string { return filepath.Join("testdata", "glpk", name+".lp") }

func linearModel() *lpgen.LPModel {
	return lpgen.NewLPModel().
		SetObjective("Maximize", "3 x + 2 y").
		AddConstraint("x + y <= 4").
		AddConstraint("x + 3 y <= 6")
}

// Taking b, c and g or taking d and g both reach 53, so only the objective is
// unique.
func knapsackModel() *lpgen.LPModel {
	m := lpgen.NewLPModel().
		SetObjective("Maximize", "10 a + 13 b + 18 c + 31 d + 7 e + 15 f + 22 g + 9 h").
		AddConstraint("11 a + 15 b + 20 c + 35 d + 10 e + 33 f + 24 g + 12 h <= 60")
	for _, v := range []string{"a", "b", "c", "d", "e", "f", "g", "h"} {
		m.AddBinaryVar(v)
	}
	return m
}

func TestSolveWithTheDefaultEngine(t *testing.T) {
	tests := []struct {
		name      string
		model     *lpgen.LPModel
		status    Status
		objective float64
		values    map[string]float64
	}{
		{name: "linear", model: linearModel(), status: StatusOptimal, objective: 12,
			values: map[string]float64{"x": 4, "y": 0}},
		{name: "binary knapsack", model: knapsackModel(), status: StatusOptimal, objective: 53},
		{name: "infeasible", model: lpgen.NewLPModel().SetObjective("Minimize", "x + y").
			AddConstraint("x + y >= 10").AddConstraint("x + y <= 5"), status: StatusInfeasible},
		{name: "unbounded", model: lpgen.NewLPModel().SetObjective("Maximize", "x + y").
			AddConstraint("x - y <= 1"), status: StatusUnbounded},
		{name: "integer infeasible", model: lpgen.NewLPModel().SetObjective("Maximize", "x").
			AddConstraint("2 x = 1").AddBound("0 <= x <= 10").AddIntegerVar("x"), status: StatusInfeasible},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sol, err := Solve(tt.model, Options{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if sol == nil {
				t.Fatal("a nil error came with a nil solution")
			}
			if sol.Engine != EngineMILP {
				t.Errorf("engine %q, want %q", sol.Engine, EngineMILP)
			}
			if sol.Status != tt.status {
				t.Fatalf("status %v, want %v", sol.Status, tt.status)
			}
			if tt.status != StatusOptimal {
				if sol.Values != nil {
					t.Errorf("values %v, want nil for %v", sol.Values, tt.status)
				}
				return
			}
			if math.Abs(sol.Objective-tt.objective) > 1e-9 {
				t.Errorf("objective %v, want %v", sol.Objective, tt.objective)
			}
			for name, want := range tt.values {
				if got, ok := sol.Values[name]; !ok || math.Abs(got-want) > 1e-9 {
					t.Errorf("%s = %v (present %v), want %v", name, got, ok, want)
				}
			}
		})
	}
}

func TestSolveRefusesAModelItCannotSolve(t *testing.T) {
	quietLP(t)
	tests := []struct {
		name  string
		model *lpgen.LPModel
	}{
		{"no model", nil},
		{"no objective", lpgen.NewLPModel().AddConstraint("x <= 1")},
		{"an unknown objective type", lpgen.NewLPModel().SetObjective("upwards", "x").AddConstraint("x <= 1")},
		{"a syntax error", lpgen.NewLPModel().SetObjective("min", "x").AddConstraint("x >=")},
		{"crossed bounds", lpgen.NewLPModel().SetObjective("min", "x").AddConstraint("x >= 1").AddBound("5 <= x <= 3")},
		{"a fractional bound on an integer", lpgen.NewLPModel().SetObjective("min", "x").
			AddConstraint("x >= 0").AddBound("0.5 <= x <= 0.7").AddIntegerVar("x")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sol, err := Solve(tt.model, Options{})
			if sol != nil {
				t.Errorf("got a solution %+v alongside the error", sol)
			}
			if !errors.Is(err, ErrInvalidModel) {
				t.Errorf("error %v, want ErrInvalidModel", err)
			}
		})
	}
}

func TestSolveFile(t *testing.T) {
	quietLP(t)

	sol, err := SolveFile(fixtureLP("lp_optimal"), Options{})
	if err != nil || sol.Status != StatusOptimal || math.Abs(sol.Objective-12) > 1e-9 {
		t.Fatalf("lp_optimal.lp: solution %+v, error %v; want optimal 12", sol, err)
	}

	sol, err = SolveFile(fixtureLP("syntax_error"), Options{})
	if sol != nil || !errors.Is(err, ErrInvalidModel) {
		t.Errorf("syntax_error.lp: solution %+v, error %v; want ErrInvalidModel", sol, err)
	}
	if err != nil && !strings.Contains(err.Error(), "6") {
		t.Errorf("the error %q does not name line 6, where the right-hand side is missing", err)
	}

	sol, err = SolveFile(filepath.Join(t.TempDir(), "missing.lp"), Options{})
	if sol != nil || !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a missing file: solution %+v, error %v; want fs.ErrNotExist", sol, err)
	}
}

// The 400-variable knapsack GLPK could not finish in two seconds.
func TestSolveStopsAtTheTimeLimit(t *testing.T) {
	start := time.Now()
	sol, err := SolveFile(fixtureLP("mip_time_limit_feasible"), Options{TimeLimit: 200 * time.Millisecond})
	if err != nil {
		t.Fatalf("a time limit is an outcome, not an error: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 10*time.Second {
		t.Errorf("the solve took %v with a 200ms limit", elapsed)
	}
	switch sol.Status {
	case StatusFeasible, StatusOptimal:
		if len(sol.Values) != 400 {
			t.Errorf("%v with %d values, want 400", sol.Status, len(sol.Values))
		}
	default:
		t.Errorf("status %v, want feasible (or optimal if it finished)", sol.Status)
	}
}

// An equality-constrained 0/1 problem GLPK found no solution for within one
// second.
func TestSolveStopsBeforeFindingASolution(t *testing.T) {
	sol, err := SolveFile(fixtureLP("mip_time_limit_no_solution"), Options{TimeLimit: 20 * time.Millisecond})
	if err != nil {
		t.Fatalf("a time limit is an outcome, not an error: %v", err)
	}
	if sol.Values != nil {
		t.Errorf("status %v came with values; want none before a solution is found", sol.Status)
	}
	if sol.Status != StatusStopped && sol.Status != StatusInfeasible {
		t.Errorf("status %v, want stopped", sol.Status)
	}
}

func TestSolveLeavesTheEnvironmentAlone(t *testing.T) {
	t.Setenv("GLPK_PATH", "untouched")
	path := os.Getenv("PATH")
	if _, err := Solve(linearModel(), Options{}); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("PATH") != path || os.Getenv("GLPK_PATH") != "untouched" {
		t.Errorf("Solve changed the environment: PATH %q, GLPK_PATH %q", os.Getenv("PATH"), os.Getenv("GLPK_PATH"))
	}
}

func TestGLPKEngineWithoutGLPSOL(t *testing.T) {
	quietLP(t)
	t.Setenv("GLPK_PATH", filepath.Join(t.TempDir(), "missing"))
	t.Setenv("PATH", t.TempDir())

	sol, err := Solve(linearModel(), Options{Engine: EngineGLPK})
	if sol != nil || !errors.Is(err, ErrEngineUnavailable) {
		t.Errorf("solution %+v, error %v; want ErrEngineUnavailable", sol, err)
	}
}

func TestSolveRefusesAnUnknownEngine(t *testing.T) {
	sol, err := Solve(linearModel(), Options{Engine: "cplex"})
	if sol != nil || !errors.Is(err, ErrEngineUnavailable) {
		t.Errorf("solution %+v, error %v; want ErrEngineUnavailable", sol, err)
	}
}

func glpsolOrSkip(t *testing.T) string {
	t.Helper()
	path, err := exec.LookPath("glpsol")
	if err != nil {
		t.Skip("glpsol is not installed")
	}
	return path
}

func TestGLPKEngineFindsGLPSOLThroughGLPKPath(t *testing.T) {
	glpsol := glpsolOrSkip(t)
	for _, glpkPath := range []string{glpsol, filepath.Dir(glpsol)} {
		t.Run(glpkPath, func(t *testing.T) {
			t.Setenv("GLPK_PATH", glpkPath)
			t.Setenv("PATH", t.TempDir())
			sol, err := Solve(linearModel(), Options{Engine: EngineGLPK})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if sol.Engine != EngineGLPK || sol.Status != StatusOptimal || sol.Objective != 12 {
				t.Errorf("solution %+v, want GLPK optimal 12", sol)
			}
			if !strings.Contains(sol.Log, "OPTIMAL") {
				t.Errorf("the log does not carry glpsol's report:\n%s", sol.Log)
			}
		})
	}
}

func TestGLPKEngineStopsAtTheTimeLimit(t *testing.T) {
	glpsolOrSkip(t)
	sol, err := SolveFile(fixtureLP("mip_time_limit_feasible"), Options{Engine: EngineGLPK, TimeLimit: time.Second})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sol.Status != StatusFeasible || len(sol.Values) != 400 {
		t.Errorf("status %v with %d values, want feasible with 400", sol.Status, len(sol.Values))
	}
}

// Every recorded model, solved by both engines, must agree on the outcome and
// on the objective value.
func TestEnginesAgree(t *testing.T) {
	glpsolOrSkip(t)
	for _, name := range []string{
		"lp_optimal", "mip_optimal", "long_names", "lp_infeasible", "lp_unbounded",
		"mip_infeasible", "mip_unbounded", "solvemodel_format", "lpgen_format",
	} {
		t.Run(name, func(t *testing.T) {
			milp, err := SolveFile(fixtureLP(name), Options{})
			if err != nil {
				t.Fatalf("go-milp: %v", err)
			}
			glpk, err := SolveFile(fixtureLP(name), Options{Engine: EngineGLPK})
			if err != nil {
				t.Fatalf("GLPK: %v", err)
			}
			if milp.Status != glpk.Status {
				t.Fatalf("go-milp says %v, GLPK says %v", milp.Status, glpk.Status)
			}
			if milp.Status == StatusOptimal && math.Abs(milp.Objective-glpk.Objective) > 1e-6 {
				t.Errorf("go-milp objective %v, GLPK objective %v", milp.Objective, glpk.Objective)
			}
		})
	}
}

func TestSolutionToDataTable(t *testing.T) {
	sol := &Solution{Status: StatusOptimal, Objective: 12,
		Values: map[string]float64{"y": 0, "x": 4}, order: []string{"x", "y"}}
	dt := sol.ToDataTable()
	if got := dt.GetColByName("Variable").Data(); !reflect.DeepEqual(got, []any{"x", "y"}) {
		t.Errorf("Variable column %v, want [x y]", got)
	}
	if got := dt.GetColByName("Value").Data(); !reflect.DeepEqual(got, []any{4.0, 0.0}) {
		t.Errorf("Value column %v, want [4 0]", got)
	}

	empty := (&Solution{Status: StatusInfeasible}).ToDataTable()
	if empty == nil {
		t.Fatal("ToDataTable returned nil")
	}
	if rows, cols := empty.Size(); rows != 0 || cols != 2 {
		t.Errorf("an infeasible solution gave %d rows and %d columns, want 0 and 2", rows, cols)
	}
	if err := empty.Err(); err != nil {
		t.Errorf("no solution is an outcome, not a failure: %v", err)
	}
}

func TestStatusString(t *testing.T) {
	want := map[Status]string{
		StatusOptimal: "optimal", StatusFeasible: "feasible", StatusInfeasible: "infeasible",
		StatusUnbounded: "unbounded", StatusStopped: "stopped",
	}
	for status, text := range want {
		if status.String() != text {
			t.Errorf("%d.String() = %q, want %q", int(status), status.String(), text)
		}
	}
}
