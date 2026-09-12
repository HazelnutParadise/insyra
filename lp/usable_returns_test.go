package lp

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/lpgen"
)

// Docs/lp.md's example calls result.Show() on the first return, and every
// failure path used to hand back nil, which panics there.
//
// Only the paths that return before initGLPK are exercised end to end. initGLPK
// downloads and compiles GLPK when glpsol is not on PATH (LP-1, #257), so a
// test must not reach it — which is also why the paths that need a solver are
// tested through their pieces instead.

// mustBeUsable does what the documented example does with both returns.
func mustBeUsable(t *testing.T, what string, result, info *insyra.DataTable) {
	t.Helper()
	if result == nil {
		t.Fatalf("%s: the result table is nil", what)
	}
	if info == nil {
		t.Fatalf("%s: the info table is nil", what)
	}
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("%s: using the returned tables panicked: %v", what, r)
		}
	}()
	result.Show()
	info.Show()
	if err := result.ToCSV(filepath.Join(t.TempDir(), "out.csv"), false, false, false); err != nil {
		t.Fatalf("%s: writing the result table: %v", what, err)
	}
}

func TestSolveFromFile_TooManyTimeouts(t *testing.T) {
	quietLP(t)

	result, info := SolveFromFile("model.lp", 1, 2)
	mustBeUsable(t, "two timeouts", result, info)

	if result.Err() == nil {
		t.Fatal("the result table carries no error")
	}
	if !strings.Contains(result.Err().Error(), "only one timeout") {
		t.Errorf("the error %q does not explain the problem", result.Err())
	}
	if got := statusOf(t, info); got != "Error" {
		t.Errorf("info Status: got %q, want \"Error\"", got)
	}
}

func TestSolveModel_RejectsNilModelAndBadArguments(t *testing.T) {
	quietLP(t)

	t.Run("nil model", func(t *testing.T) {
		result, info := SolveModel(nil)
		mustBeUsable(t, "nil model", result, info)
		if result.Err() == nil {
			t.Fatal("the result table carries no error")
		}
		if !strings.Contains(result.Err().Error(), "no model") {
			t.Errorf("the error %q does not say the model was missing", result.Err())
		}
	})

	t.Run("two timeouts", func(t *testing.T) {
		model := lpgen.NewLPModel().SetObjective("Maximize", "x").AddConstraint("x <= 1")
		result, info := SolveModel(model, 1, 2)
		mustBeUsable(t, "two timeouts", result, info)
		if result.Err() == nil {
			t.Fatal("the result table carries no error")
		}
	})
}

// parseGLPKOutputFromFile's return is handed straight back to the caller as the
// first result, so its nil was the nil the caller saw.
func TestParseGLPKOutputFromFile_UnreadableFileIsUsable(t *testing.T) {
	quietLP(t)

	dt := parseGLPKOutputFromFile(filepath.Join(t.TempDir(), "nope.txt"))
	if dt == nil {
		t.Fatal("a missing solution file gave nil")
	}
	if dt.Err() == nil {
		t.Error("the returned table carries no error")
	}
	if rows, _ := dt.Size(); rows != 0 {
		t.Errorf("the returned table has %d rows, want 0", rows)
	}
	dt.Show() // must not panic
}

// A failure before the solver runs still produces a complete info table.
func TestFailedPair(t *testing.T) {
	quietLP(t)

	result, info := failedPair("SolveFromFile", "something went wrong: %d", 42)

	if result.Err() == nil || !strings.Contains(result.Err().Error(), "something went wrong: 42") {
		t.Errorf("result error: got %v", result.Err())
	}
	if got := statusOf(t, info); got != "Error" {
		t.Errorf("info Status: got %q, want \"Error\"", got)
	}
	if got := rowOf(t, info, "Warnings"); got != "something went wrong: 42" {
		t.Errorf("info Warnings: got %q", got)
	}
}

func TestJoinWarnings(t *testing.T) {
	tests := []struct {
		existing, extra, want string
	}{
		{existing: "", extra: "b", want: "b"},
		{existing: "a", extra: "b", want: "a; b"},
		{existing: "warning: x", extra: "cannot read", want: "warning: x; cannot read"},
	}
	for _, tt := range tests {
		if got := joinWarnings(tt.existing, tt.extra); got != tt.want {
			t.Errorf("joinWarnings(%q, %q) = %q, want %q", tt.existing, tt.extra, got, tt.want)
		}
	}
}

func statusOf(t *testing.T, info *insyra.DataTable) string {
	t.Helper()
	return rowOf(t, info, "Status")
}

func rowOf(t *testing.T, info *insyra.DataTable, name string) string {
	t.Helper()
	col := info.GetColByName("Additional Info")
	if col == nil {
		t.Fatal("the info table has no \"Additional Info\" column")
	}
	for i, n := range info.RowNames() {
		if n == name {
			s, _ := col.Get(i).(string)
			return s
		}
	}
	t.Fatalf("the info table has no %q row", name)
	return ""
}

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
