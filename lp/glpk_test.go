package lp

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// The files under testdata/glpk were recorded with GLPK 5.0 on 2026-09-13:
//
//	glpsol --lp NAME.lp --output NAME.report.txt --write NAME.write.txt > NAME.stdout.txt
//
// (the two time-limit cases add --tmlim). readGLPKResult works from those
// three files alone, so nothing in this file runs glpsol.
func glpkFixture(t *testing.T, name string) (report, solution, output string) {
	t.Helper()
	read := func(suffix string) string {
		b, err := os.ReadFile(filepath.Join("testdata", "glpk", name+suffix))
		if err != nil {
			t.Fatalf("fixture %s%s: %v", name, suffix, err)
		}
		return string(b)
	}
	return read(".report.txt"), read(".write.txt"), read(".stdout.txt")
}

func TestReadGLPKResult(t *testing.T) {
	tests := []struct {
		name      string
		status    Status
		objective float64
		values    map[string]float64
		order     []string
	}{
		{name: "lp_optimal", status: StatusOptimal, objective: 12,
			values: map[string]float64{"x": 4, "y": 0}, order: []string{"x", "y"}},
		{name: "mip_optimal", status: StatusOptimal, objective: 53,
			values: map[string]float64{"a": 0, "b": 1, "c": 1, "d": 0, "e": 0, "f": 0, "g": 1, "h": 0},
			order:  []string{"a", "b", "c", "d", "e", "f", "g", "h"}},
		// The report prints the second value as 123457 and puts each name on
		// a line of its own; the value comes from the --write file instead.
		{name: "long_names", status: StatusOptimal, objective: 246913.578,
			values: map[string]float64{"production_of_widgets": 0, "another_quite_long_variable_name": 123456.789},
			order:  []string{"production_of_widgets", "another_quite_long_variable_name"}},
		{name: "solvemodel_format", status: StatusOptimal, objective: 27,
			values: map[string]float64{"x1": 4, "x2": 3, "x3": 0, "x4": 0}, order: []string{"x1", "x2", "x3", "x4"}},
		{name: "lpgen_format", status: StatusOptimal, objective: 27,
			values: map[string]float64{"x1": 4, "x2": 3, "x3": 0, "x4": 0}, order: []string{"x1", "x2", "x3", "x4"}},
		// With presolve on, GLPK writes an infeasible or unbounded LP with
		// status letters "u u"; only its message tells the two apart.
		{name: "lp_infeasible", status: StatusInfeasible},
		{name: "lp_unbounded", status: StatusUnbounded},
		{name: "mip_infeasible", status: StatusInfeasible},
		{name: "mip_unbounded", status: StatusUnbounded},
		{name: "mip_time_limit_no_solution", status: StatusStopped},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sol, err := readGLPKResult(glpkFixture(t, tt.name))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if sol.Status != tt.status {
				t.Errorf("status %v, want %v", sol.Status, tt.status)
			}
			if tt.values == nil {
				if sol.Values != nil {
					t.Errorf("values %v, want nil for %v", sol.Values, tt.status)
				}
				return
			}
			if sol.Objective != tt.objective {
				t.Errorf("objective %v, want %v", sol.Objective, tt.objective)
			}
			if !reflect.DeepEqual(sol.Values, tt.values) {
				t.Errorf("values %v, want %v", sol.Values, tt.values)
			}
			if !reflect.DeepEqual(sol.order, tt.order) {
				t.Errorf("variable order %v, want %v", sol.order, tt.order)
			}
		})
	}
}

// GLPK stopped at --tmlim 2 with an incumbent: the status letter is "f" and
// every column still has a value.
func TestReadGLPKResultTimeLimitWithASolution(t *testing.T) {
	report, solution, output := glpkFixture(t, "mip_time_limit_feasible")
	sol, err := readGLPKResult(report, solution, output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sol.Status != StatusFeasible {
		t.Errorf("status %v, want %v", sol.Status, StatusFeasible)
	}
	if sol.Objective != 12874 {
		t.Errorf("objective %v, want 12874", sol.Objective)
	}
	if len(sol.Values) != 400 || sol.Values["x2"] != 1 || sol.Values["x0"] != 0 {
		t.Errorf("got %d values, x0=%v x2=%v; want 400 values with x0=0 and x2=1",
			len(sol.Values), sol.Values["x0"], sol.Values["x2"])
	}
	if !strings.Contains(sol.Log, "TIME LIMIT EXCEEDED") || !strings.Contains(sol.Log, "INTEGER NON-OPTIMAL") {
		t.Errorf("the log does not carry glpsol's output and report:\n%s", sol.Log)
	}
}

// Without presolve GLPK does write a final status; those letters must be
// read the same way.
func TestReadGLPKResultWithoutPresolve(t *testing.T) {
	tests := []struct {
		name     string
		solution string
		status   Status
	}{
		{"infeasible", "c\ns bas 2 2 n i 5\ne o f\n", StatusInfeasible},
		{"unbounded", "c\ns bas 1 2 f n 1\ne o f\n", StatusUnbounded},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sol, err := readGLPKResult("", tt.solution, "")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if sol.Status != tt.status {
				t.Errorf("status %v, want %v", sol.Status, tt.status)
			}
		})
	}
}

// GLPK refuses these bounds only when it starts solving. glpsol still exits 0
// and writes an undefined status; the message is the only sign.
func TestReadGLPKResultRefusedBounds(t *testing.T) {
	for _, name := range []string{"bounds_crossed", "integer_bound_fraction"} {
		t.Run(name, func(t *testing.T) {
			sol, err := readGLPKResult(glpkFixture(t, name))
			if sol != nil || !errors.Is(err, ErrInvalidModel) {
				t.Errorf("solution %+v, error %v; want ErrInvalidModel", sol, err)
			}
		})
	}
}

func TestReadGLPKResultFailures(t *testing.T) {
	tests := []struct {
		name             string
		solution, output string
	}{
		{"an undefined status with no message that explains it", "c\ns bas 1 1 u u 0\ne o f\n", "something unexpected\n"},
		{"an empty solution file", "", ""},
		{"a solution line that cannot be read", "s mip\n", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sol, err := readGLPKResult("", tt.solution, tt.output)
			if sol != nil {
				t.Errorf("got a solution %+v, want nil", sol)
			}
			if !errors.Is(err, ErrSolverFailed) {
				t.Errorf("error %v, want ErrSolverFailed", err)
			}
		})
	}
}
