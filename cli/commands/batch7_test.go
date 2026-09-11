package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/cli/env"
)

// tableCtx returns a context holding a three-column table `dt` and a list `x`.
func tableCtx(t *testing.T) *ExecContext {
	t.Helper()
	ctx := newTestExecContext(t)
	ctx.Vars["dt"] = insyra.NewDataTable(
		insyra.NewDataList(1, 2, 3).SetName("price"),
		insyra.NewDataList(4, 5, 6).SetName("qty"),
		insyra.NewDataList("a", "b", "c").SetName("tag"),
	)
	ctx.Vars["x"] = insyra.NewDataList(1.0, 2.0, 3.0)
	return ctx
}

// CLI-7: a command that could not do what it was asked must say so. Reporting
// success with exit code 0 makes a broken script look like a working one.
func TestCommandsReportFailureInsteadOfSuccess(t *testing.T) {
	cases := map[string][]string{
		"sort by a missing column":        {"sort", "dt", "nonexistent"},
		"dropcol a missing column":        {"dropcol", "dt", "nonexistent"},
		"droprow out of range":            {"droprow", "dt", "99"},
		"swap missing columns":            {"swap", "dt", "col", "nope1", "nope2"},
		"ccl that cannot compile":         {"ccl", "dt", "bogus((("},
		"addcolccl that cannot compile":   {"addcolccl", "dt", "newc", "nonsense((("},
		"filter with a broken expression": {"filter", "dt", "bogus((("},
	}
	for name, argv := range cases {
		t.Run(name, func(t *testing.T) {
			ctx := tableCtx(t)
			err := Dispatch(ctx, argv[0], argv[1:])
			if err == nil {
				t.Fatalf("%v reported success; output was %q", argv, outputOf(ctx))
			}
		})
	}
}

// CLI-8: a misspelled option value must be rejected, not silently replaced by
// the default — a statistical result computed under the wrong assumption is
// indistinguishable from a right one.
func TestMisspelledEnumValuesAreRejected(t *testing.T) {
	cases := map[string][]string{
		"sort direction":    {"sort", "dt", "price", "dsc"},
		"ttest variance":    {"ttest", "two", "x", "x", "eqaul"},
		"ztest alternative": {"ztest", "single", "x", "3", "1", "bogus"},
		"clean stddev":      {"clean", "x", "outliers", "abc"},
		"merge extra token": {"merge", "dt", "dt", "horizontal", "inner", "junk"},
	}
	for name, argv := range cases {
		t.Run(name, func(t *testing.T) {
			ctx := tableCtx(t)
			err := Dispatch(ctx, argv[0], argv[1:])
			if err == nil {
				t.Fatalf("%v was accepted; output was %q", argv, outputOf(ctx))
			}
		})
	}

	// The correct spellings still work.
	for _, argv := range [][]string{
		{"sort", "dt", "price", "desc"},
		{"sort", "dt", "price", "asc"},
		{"clean", "x", "outliers", "2.5"},
	} {
		ctx := tableCtx(t)
		if err := Dispatch(ctx, argv[0], argv[1:]); err != nil {
			t.Errorf("%v was rejected: %v", argv, err)
		}
	}
}

// CLI-9: config must not accept a key it does not know or a value it cannot
// use, and must not persist either.
func TestConfigRejectsUnknownKeysAndValues(t *testing.T) {
	base := t.TempDir()
	mgr := env.NewManager(base, "envs")
	if err := mgr.EnsureDefaultEnvironment(); err != nil {
		t.Fatal(err)
	}
	ctx := newTestExecContext(t)
	ctx.Env = mgr
	ctx.EnvName = "default"

	for name, argv := range map[string][]string{
		"unknown key":    {"bogus-key", "123"},
		"bad log level":  {"log-level", "nonsense"},
		"bad boolean":    {"no-color", "maybe"},
		"bad accel mode": {"accel-mode", "turbo"},
	} {
		t.Run(name, func(t *testing.T) {
			if err := runConfigCommand(ctx, argv); err == nil {
				t.Fatalf("config %v was accepted", argv)
			}
		})
	}

	for _, argv := range [][]string{
		{"log-level", "debug"},
		{"no-color", "true"},
		{"accel-mode", "cpu"},
		{"default-env", "scratch"},
	} {
		if err := runConfigCommand(ctx, argv); err != nil {
			t.Errorf("config %v was rejected: %v", argv, err)
		}
	}
}

// CLI-10: the usage text must describe the command that exists.
func TestAccelUsageMatchesTheCommand(t *testing.T) {
	registryMu.RLock()
	handler := Registry["accel"]
	registryMu.RUnlock()
	if handler == nil {
		t.Fatal("accel is not registered")
	}
	if strings.Contains(handler.Usage, "run") {
		t.Errorf("Usage claims an `accel run` subcommand that does not exist: %q", handler.Usage)
	}
	// --precision chose the precision of `accel run`, removed in v0.3.1. No
	// action reads it, so advertising it promised something that never happens.
	if strings.Contains(handler.Usage, "--precision") {
		t.Errorf("Usage advertises --precision, which nothing reads: %q", handler.Usage)
	}

	ctx := newTestExecContext(t)
	if err := runAccelCommand(ctx, []string{"run"}); err == nil {
		t.Error("`accel run` was accepted")
	}
}

// accel registers the one flag it reads, --mode, so one-shot mode and the REPL
// accept the same arguments and Cobra rejects anything else before it runs.
func TestAccelFlagsAreRegistered(t *testing.T) {
	ctx := newTestExecContext(t)
	for _, cmd := range BuildCobraCommands(ctx) {
		if cmd.Name() != "accel" {
			continue
		}
		if cmd.Flags().Lookup("mode") == nil {
			t.Error("accel does not register --mode")
		}
		if cmd.Flags().Lookup("precision") != nil {
			t.Error("accel registers --precision, which nothing reads")
		}
		return
	}
	t.Fatal("accel command not built")
}

// CLI-13: an argument the command does not understand must be reported, not
// dropped. Silently ignoring it makes a typo look like it worked.
func TestUnknownTokensAreRejected(t *testing.T) {
	ctx := tableCtx(t)
	ctx.Vars["series"] = insyra.NewDataList(1.0, 2.0, 3.0)
	if err := Dispatch(ctx, "plot", []string{"line", "series", "extra", "junk"}); err == nil {
		t.Fatalf("plot accepted unknown tokens; output was %q", outputOf(ctx))
	}

	registryMu.RLock()
	handler := Registry["plot"]
	registryMu.RUnlock()
	if strings.Contains(handler.Usage, "options...") {
		t.Errorf("Usage advertises options plot does not accept: %q", handler.Usage)
	}
}

// CLI-14: an argument outside the valid range must be an error, not an empty
// result reported as a success.
func TestOutOfRangeArgumentsAreRejected(t *testing.T) {
	for name, argv := range map[string][]string{
		"sample size zero":        {"sample", "x", "0"},
		"sample size negative":    {"sample", "x", "-1"},
		"sample larger than list": {"sample", "x", "99"},
		"too few column names":    {"setcolnames", "dt", "only_one"},
		"too many column names":   {"setcolnames", "dt", "a", "b", "c", "d"},
	} {
		t.Run(name, func(t *testing.T) {
			ctx := tableCtx(t)
			if err := Dispatch(ctx, argv[0], argv[1:]); err == nil {
				t.Fatalf("%v was accepted; output was %q", argv, outputOf(ctx))
			}
		})
	}

	// The valid forms still work.
	ctx := tableCtx(t)
	if err := Dispatch(ctx, "sample", []string{"x", "2", "as", "s"}); err != nil {
		t.Errorf("sample x 2: %v", err)
	}
	ctx = tableCtx(t)
	if err := Dispatch(ctx, "setcolnames", []string{"dt", "a", "b", "c"}); err != nil {
		t.Errorf("setcolnames with the right count: %v", err)
	}
}

// outputOf returns what a command wrote to the context's buffered output.
func outputOf(ctx *ExecContext) string {
	if buf, ok := ctx.Output.(*bytes.Buffer); ok {
		return buf.String()
	}
	return ""
}

// CLI-13, the other half: fetch takes parameters only for `news`.
func TestFetchYahooRejectsExtraParams(t *testing.T) {
	ctx := newTestExecContext(t)
	err := Dispatch(ctx, "fetch", []string{"yahoo", "AAPL", "quote", "extra"})
	if err == nil {
		t.Fatal("fetch yahoo AAPL quote extra was accepted")
	}
	if !strings.Contains(err.Error(), "extra") {
		t.Fatalf("the error should name the unexpected argument: %v", err)
	}
}
