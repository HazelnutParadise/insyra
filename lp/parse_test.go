package lp

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// Everything below runs without glpsol on PATH. SolveFromFile and SolveModel
// need the solver, but the four functions that decide what a caller actually
// receives are file and string handling, and they had no test.

// writeSolution puts content in a temporary file and returns its path.
func writeSolution(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "solution.txt")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("writing the fixture: %v", err)
	}
	return path
}

// lines returns the parsed solution as one string per output row.
func lines(t *testing.T, path string) []string {
	t.Helper()
	dt := parseGLPKOutputFromFile(path)
	if dt == nil {
		t.Fatal("parseGLPKOutputFromFile returned nil")
	}
	rows, cols := dt.Size()
	if rows == 0 {
		return nil
	}
	if cols != 1 {
		t.Fatalf("expected one column of lines, got %d columns", cols)
	}
	col := dt.GetColByNumber(0)
	out := make([]string, 0, rows)
	for i := 0; i < rows; i++ {
		s, ok := col.Get(i).(string)
		if !ok {
			t.Fatalf("row %d is %T, want string", i, col.Get(i))
		}
		out = append(out, s)
	}
	return out
}

// The two header labels are rewritten so a reader can tell which is which:
// GLPK's "Rows" are constraints and its "Columns" are variables.
func TestParseGLPKOutputFromFile_RenamesRowsAndColumns(t *testing.T) {
	path := writeSolution(t, "Problem:    demo\nRows:       3\nColumns:    2\nStatus:     OPTIMAL\n")

	got := lines(t, path)
	want := []string{
		"Problem:    demo",
		"Rows(Constraints):       3",
		"Columns(Variables):    2",
		"Status:     OPTIMAL",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

// Blank lines are dropped and each kept line is trimmed, so the separator blank
// line GLPK writes before the variable table does not become an empty row.
func TestParseGLPKOutputFromFile_SkipsBlanksAndTrims(t *testing.T) {
	path := writeSolution(t, "\n   Objective:  obj = 22 (MAXimum)   \n\n\n     1 x            B             4   \n\n")

	got := lines(t, path)
	want := []string{
		"Objective:  obj = 22 (MAXimum)",
		"1 x            B             4",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestParseGLPKOutputFromFile_EmptyFile(t *testing.T) {
	path := writeSolution(t, "")

	dt := parseGLPKOutputFromFile(path)
	if dt == nil {
		t.Fatal("an empty solution file gave nil, want an empty DataTable")
	}
	if rows, _ := dt.Size(); rows != 0 {
		t.Errorf("an empty solution file produced %d rows, want 0", rows)
	}
}

// A missing file is how a timed-out or crashed solver shows up. The function
// reports it and hands back nil, which is what SolveFromFile then returns as
// its result table.
func TestParseGLPKOutputFromFile_MissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.txt")

	if dt := parseGLPKOutputFromFile(path); dt != nil {
		t.Errorf("a missing file gave %v, want nil", dt)
	}
}

func TestExtractIterationNodeCounts(t *testing.T) {
	tests := []struct {
		name       string
		output     string
		iterations string
		nodes      string
	}{
		{
			name:       "no output",
			output:     "",
			iterations: "",
			nodes:      "",
		},
		{
			name:       "simplex only",
			output:     "*     0: obj =   0.0\n*     4: obj =  22.0\n",
			iterations: "4",
			nodes:      "",
		},
		{
			name:       "branch and bound",
			output:     "*     4: obj =  22.0\n+     7: mip =     not found yet\n+    19: >>>>>  20.0\n",
			iterations: "4",
			nodes:      "19",
		},
		{
			name:       "lines that only look similar",
			output:     "GLPK Simplex Optimizer\nsomething: 12\n",
			iterations: "",
			nodes:      "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iterations, nodes := extractIterationNodeCounts(tt.output)
			if iterations != tt.iterations {
				t.Errorf("iterations: got %q, want %q", iterations, tt.iterations)
			}
			if nodes != tt.nodes {
				t.Errorf("nodes: got %q, want %q", nodes, tt.nodes)
			}
		})
	}
}

func TestExtractWarnings(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{name: "none", output: "Reading problem data...\nOPTIMAL SOLUTION FOUND\n", want: ""},
		{
			name:   "one",
			output: "warning: objective scaling required\nOPTIMAL\n",
			want:   "warning: objective scaling required",
		},
		{
			name:   "several, in order",
			output: "warning: first\nnoise\nwarning: second\n",
			want:   "warning: first; warning: second",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractWarnings([]byte(tt.output)); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// info_order_test.go pins the row order. This pins what is in the rows.
func TestCreateAdditionalInfoDataTable_Values(t *testing.T) {
	dt := createAdditionalInfoDataTable("Success", 1.5, "warning: w", "full output", "12", "3")

	col := dt.GetColByName("Additional Info")
	if col == nil {
		t.Fatal("the \"Additional Info\" column is missing")
	}
	want := []any{"Success", "1.50 seconds", "warning: w", "full output", "12", "3"}
	if got := col.Data(); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
