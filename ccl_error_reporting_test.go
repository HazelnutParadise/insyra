package insyra

import (
	"errors"
	"strings"
	"testing"

	engineccl "github.com/HazelnutParadise/insyra/engine/ccl"
	"github.com/HazelnutParadise/insyra/internal/ccl"
)

// CCL-16: the caller's first question is "is my formula wrong, or my data?".
// Both failures used to share the prefix "Failed to apply CCL on DataTable".
func TestCCLCompileAndEvalFailuresAreDistinguishable(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)

	dt := NewDataTable(
		NewDataList(10, 20, 30).SetName("price"),
		NewDataList(1, 0, 3).SetName("qty"),
	)
	dt.AddColUsingCCL("bad", "SUM(A B)")
	compileErr := dt.PopErr()
	if compileErr == nil {
		t.Fatal("SUM(A B) was accepted")
	}
	var ce *ccl.CompileError
	if !errors.As(compileErr, &ce) {
		t.Fatalf("errors.As could not reach a *ccl.CompileError: %v", compileErr)
	}
	if ce.Expr != "SUM(A B)" {
		t.Errorf("CompileError.Expr = %q", ce.Expr)
	}

	dt2 := NewDataTable(
		NewDataList(10, 20, 30).SetName("price"),
		NewDataList(1, 0, 3).SetName("qty"),
	)
	dt2.AddColUsingCCL("bad", "A / B")
	evalErr := dt2.PopErr()
	if evalErr == nil {
		t.Fatal("A / B on a zero was accepted")
	}
	var ee *ccl.EvalError
	if !errors.As(evalErr, &ee) {
		t.Fatalf("errors.As could not reach a *ccl.EvalError: %v", evalErr)
	}
	if ee.Row != 1 {
		t.Errorf("EvalError.Row = %d, want 1 (the row whose qty is 0)", ee.Row)
	}

	// And the two messages must not read the same.
	if compileErr.Message == evalErr.Message {
		t.Error("a compile failure and an evaluation failure produce the same message")
	}
}

// The row must appear in the text a user reads, not only in the struct.
func TestCCLEvalFailureMessageNamesTheRow(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)

	dt := NewDataTable(
		NewDataList(10, 20, 30).SetName("price"),
		NewDataList(1, 2, 0).SetName("qty"),
	)
	dt.AddColUsingCCL("bad", "A / B")
	err := dt.PopErr()
	if err == nil {
		t.Fatal("A / B on a zero was accepted")
	}
	if !strings.Contains(err.Error(), "row 2") {
		t.Errorf("message should name row 2: %v", err)
	}
}

// The failure message must not carry a stopwatch reading in place of a cause.
func TestCCLFailureMessageHasNoElapsedTime(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)

	dt := NewDataTable(NewDataList(1, 2, 3).SetName("a"))
	dt.AddColUsingCCL("bad", "SUM(A B)")
	err := dt.PopErr()
	if err == nil {
		t.Fatal("SUM(A B) was accepted")
	}
	if strings.Contains(err.Error(), "after ") {
		t.Errorf("the elapsed time is noise on a failure: %v", err)
	}
}

// The two error types must be reachable from the exported engine package too,
// not only from internal/ccl.
func TestCCLErrorTypesAreExported(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)

	dt := NewDataTable(NewDataList(1, 2, 3).SetName("a"))
	dt.AddColUsingCCL("bad", "SUM(A B)")
	err := dt.PopErr()
	var ce *engineccl.CompileError
	if !errors.As(err, &ce) {
		t.Fatalf("errors.As could not reach engine/ccl.CompileError: %v", err)
	}
}

// CCL-15: a script is several statements, and a runtime failure in one of them
// used to report only "division by zero" — nothing said which line.
func TestExecuteCCLNamesTheFailingStatement(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)

	dt := NewDataTable(
		NewDataList(10, 20, 30).SetName("price"),
		NewDataList(1, 0, 3).SetName("qty"),
	)
	dt.ExecuteCCL("NEW('x') = A + 1\nNEW('y') = A / B\nNEW('z') = A")
	err := dt.PopErr()
	if err == nil {
		t.Fatal("the script was accepted")
	}
	if !strings.Contains(err.Error(), "NEW('y') = A / B") {
		t.Errorf("the error must name the statement that failed: %v", err)
	}
	if !strings.Contains(err.Error(), "row 1") {
		t.Errorf("the error must name the row: %v", err)
	}

	var ee *ccl.EvalError
	if !errors.As(err, &ee) {
		t.Fatalf("errors.As could not reach a *ccl.EvalError: %v", err)
	}
	if ee.Expr != "NEW('y') = A / B" {
		t.Errorf("EvalError.Expr = %q", ee.Expr)
	}
	if ee.Row != 1 {
		t.Errorf("EvalError.Row = %d, want 1", ee.Row)
	}
}
