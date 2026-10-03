package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra/internal/dsl/env"
	"github.com/HazelnutParadise/insyra/stats"
)

// newSaveStateTestContext returns a context whose environment is a throwaway
// directory, holding one variable the environment can store (x) and one it
// cannot (r, a regression result).
func newSaveStateTestContext(t *testing.T) (*ExecContext, *env.Manager) {
	t.Helper()
	mgr := env.NewManager(t.TempDir(), "")
	if err := mgr.EnsureDefaultEnvironment(); err != nil {
		t.Fatalf("EnsureDefaultEnvironment: %v", err)
	}
	ctx := newTestExecContext(t)
	ctx.Env = mgr
	ctx.EnvName = "default"
	ctx.Vars["r"] = &stats.LinearRegressionResult{}
	ctx.Vars["x"] = 1.5
	return ctx, mgr
}

func TestSaveEnvStateWarnsOncePerVariable(t *testing.T) {
	ctx, mgr := newSaveStateTestContext(t)
	out := ctx.Output.(*bytes.Buffer)

	if err := SaveEnvState(ctx); err != nil {
		t.Fatalf("SaveEnvState: %v", err)
	}
	if got := strings.Count(out.String(), "warning:"); got != 1 {
		t.Fatalf("warnings after first save = %d, want 1; output:\n%s", got, out.String())
	}
	if !strings.Contains(out.String(), "r (*stats.LinearRegressionResult)") {
		t.Fatalf("warning does not name the variable and its type; output:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "environment default") {
		t.Fatalf("warning does not name the environment; output:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "another environment is opened") {
		t.Fatalf("warning does not say the variable is gone for good; output:\n%s", out.String())
	}

	restored, err := mgr.RestoreVariables("default")
	if err != nil {
		t.Fatalf("RestoreVariables: %v", err)
	}
	if got, ok := restored["x"]; !ok || got != float64(1.5) {
		t.Fatalf("restored x = %#v (present %v), want 1.5", got, ok)
	}
	if _, ok := restored["r"]; ok {
		t.Fatalf("restored state should not hold r, got %#v", restored["r"])
	}

	// The same variable at the same type is reported once, not once per save.
	if err := SaveEnvState(ctx); err != nil {
		t.Fatalf("second SaveEnvState: %v", err)
	}
	if got := strings.Count(out.String(), "warning:"); got != 1 {
		t.Fatalf("warnings after second save = %d, want 1; output:\n%s", got, out.String())
	}

	// A different type for the same variable is a different report.
	ctx.Vars["r"] = struct{ X int }{1}
	if err := SaveEnvState(ctx); err != nil {
		t.Fatalf("third SaveEnvState: %v", err)
	}
	if got := strings.Count(out.String(), "warning:"); got != 2 {
		t.Fatalf("warnings after third save = %d, want 2; output:\n%s", got, out.String())
	}
	if !strings.Contains(out.String(), "r (struct { X int })") {
		t.Fatalf("second warning does not name the new type; output:\n%s", out.String())
	}

	// A variable that is gone is no longer reported, and comes back into scope
	// for reporting if it cannot be stored again.
	delete(ctx.Vars, "r")
	if err := SaveEnvState(ctx); err != nil {
		t.Fatalf("fourth SaveEnvState: %v", err)
	}
	if got := strings.Count(out.String(), "warning:"); got != 2 {
		t.Fatalf("warnings after deleting r = %d, want 2; output:\n%s", got, out.String())
	}

	ctx.Vars["r"] = &stats.LinearRegressionResult{}
	if err := SaveEnvState(ctx); err != nil {
		t.Fatalf("fifth SaveEnvState: %v", err)
	}
	if got := strings.Count(out.String(), "warning:"); got != 3 {
		t.Fatalf("warnings after restoring r = %d, want 3; output:\n%s", got, out.String())
	}
}

func TestSaveEnvStateWithoutEnvironmentDoesNothing(t *testing.T) {
	ctx, _ := newSaveStateTestContext(t)
	ctx.EnvName = ""
	out := ctx.Output.(*bytes.Buffer)

	if err := SaveEnvState(ctx); err != nil {
		t.Fatalf("SaveEnvState without an environment: %v", err)
	}
	if out.Len() != 0 {
		t.Fatalf("SaveEnvState without an environment wrote %q", out.String())
	}
}

func TestSaveEnvStateReturnsWriteErrors(t *testing.T) {
	ctx, _ := newSaveStateTestContext(t)
	// ResolveEnvPath refuses a name that would leave the environments
	// directory, so the state cannot be written at all.
	ctx.EnvName = "../escape"

	err := SaveEnvState(ctx)
	if err == nil {
		t.Fatalf("expected an error for an environment that cannot be written")
	}
	if got := ctx.Output.(*bytes.Buffer).String(); strings.Contains(got, "warning:") {
		t.Fatalf("a state that could not be written must not warn about variables, got %q", got)
	}
}
