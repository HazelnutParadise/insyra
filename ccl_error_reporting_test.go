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
	quietLogs(t)

	dt := NewDataTable(
		NewDataList(10, 20, 30).SetName("price"),
		NewDataList(1, 0, 3).SetName("qty"),
	)
	dt.AddColUsingCCL("bad", "1 +)")
	compileErr := dt.PopErr()
	if compileErr == nil {
		t.Fatal("1 +) was accepted")
	}
	if compileErr.Level != LogLevelWarning {
		t.Errorf("Level = %v, want Warning like every other DataTable error", compileErr.Level)
	}
	var ce *ccl.CompileError
	if !errors.As(compileErr, &ce) {
		t.Fatalf("errors.As could not reach a *ccl.CompileError: %v", compileErr)
	}
	if ce.Expr != "1 +)" {
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
	quietLogs(t)

	for name, apply := range map[string]func(*DataTable) *DataTable{
		"AddColUsingCCL":         func(dt *DataTable) *DataTable { return dt.AddColUsingCCL("bad", "A / B") },
		"EditColByIndexUsingCCL": func(dt *DataTable) *DataTable { return dt.EditColByIndexUsingCCL("A", "A / B") },
		"EditColByNameUsingCCL":  func(dt *DataTable) *DataTable { return dt.EditColByNameUsingCCL("price", "A / B") },
	} {
		dt := NewDataTable(
			NewDataList(10, 20, 30).SetName("price"),
			NewDataList(1, 2, 0).SetName("qty"),
		)
		err := apply(dt).PopErr()
		if err == nil {
			t.Fatalf("%s: A / B on a zero was accepted", name)
		}
		if !strings.Contains(err.Error(), "row 2") {
			t.Errorf("%s: message should name row 2: %v", name, err)
		}
	}
}

// The failure message must not carry a stopwatch reading in place of a cause.
func TestCCLFailureMessageHasNoElapsedTime(t *testing.T) {
	quietLogs(t)

	dt := NewDataTable(NewDataList(1, 2, 3).SetName("a"))
	dt.AddColUsingCCL("bad", "1 +)")
	err := dt.PopErr()
	if err == nil {
		t.Fatal("1 +) was accepted")
	}
	if strings.Contains(err.Error(), "after ") {
		t.Errorf("the elapsed time is noise on a failure: %v", err)
	}
}

// The two error types must be reachable from the exported engine package too,
// not only from internal/ccl.
func TestCCLErrorTypesAreExported(t *testing.T) {
	quietLogs(t)

	dt := NewDataTable(NewDataList(1, 2, 3).SetName("a"))
	dt.AddColUsingCCL("bad", "1 +)")
	err := dt.PopErr()
	var ce *engineccl.CompileError
	if !errors.As(err, &ce) {
		t.Fatalf("errors.As could not reach engine/ccl.CompileError: %v", err)
	}
}

// CCL-15: a script is several statements, and a runtime failure in one of them
// used to report only "division by zero" — nothing said which line.
func TestExecuteCCLNamesTheFailingStatement(t *testing.T) {
	quietLogs(t)

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

	// Statements are still applied one at a time, as before: the one ahead of
	// the failure has run.
	if dt.GetColByName("x") == nil {
		t.Error("the statement before the failing one was not applied")
	}
}

// A compile failure inside a script names the statement too.
func TestExecuteCCLCompileFailureIsTyped(t *testing.T) {
	quietLogs(t)

	dt := NewDataTable(NewDataList(1, 2, 3).SetName("a"))
	dt.ExecuteCCL("NEW('x') = A + 1\nNEW('y') = (A B)")
	err := dt.PopErr()
	var ce *ccl.CompileError
	if !errors.As(err, &ce) {
		t.Fatalf("errors.As could not reach a *ccl.CompileError: %v", err)
	}
	if ce.Expr != "NEW('y') = (A B)" || ce.Offset != 14 {
		t.Errorf("CompileError = %+v, want the second statement at offset 14", ce)
	}
}
