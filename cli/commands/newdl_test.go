package commands

import (
	"bytes"
	"strings"
	"testing"

	insyra "github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/cli/env"
	"github.com/spf13/cobra"
)

// newOneShotRoot mirrors cli.NewRootCommand closely enough to reproduce the
// one-shot `insyra <command> ...` path: a root with persistent flags plus every
// registered command built through BuildCobraCommands.
func newOneShotRoot(t *testing.T) (*cobra.Command, *ExecContext, *bytes.Buffer) {
	t.Helper()
	out := &bytes.Buffer{}
	ctx := &ExecContext{
		Vars:    map[string]any{},
		Output:  out,
		EnvName: "default",
		Env:     env.NewManager(t.TempDir(), ""),
	}
	root := &cobra.Command{Use: "insyra", SilenceUsage: true, SilenceErrors: true}
	root.PersistentFlags().String("env", "default", "Environment name")
	root.PersistentFlags().Bool("no-color", false, "Disable colored output")
	root.SetOut(out)
	root.SetErr(out)
	for _, sub := range BuildCobraCommands(ctx) {
		root.AddCommand(sub)
	}
	return root, ctx, out
}

func runOneShot(t *testing.T, args ...string) (*ExecContext, string, error) {
	t.Helper()
	root, ctx, out := newOneShotRoot(t)
	root.SetArgs(args)
	err := root.Execute()
	return ctx, out.String(), err
}

func TestOneShot_NewDLAcceptsNegativeLiterals(t *testing.T) {
	ctx, _, err := runOneShot(t, "newdl", "0.01", "-0.004", "0.02", "as", "r")
	if err != nil {
		t.Fatalf("newdl with a negative literal failed: %v", err)
	}
	dl, ok := ctx.Vars["r"].(*insyra.DataList)
	if !ok {
		t.Fatalf("r = %T want *insyra.DataList", ctx.Vars["r"])
	}
	if dl.Len() != 3 {
		t.Fatalf("len = %d want 3", dl.Len())
	}
	if got := dl.Get(1); got != -0.004 {
		t.Errorf("r[1] = %#v want -0.004", got)
	}
}

func TestOneShot_AddRowAcceptsNegativeLiterals(t *testing.T) {
	root, ctx, out := newOneShotRoot(t)
	ctx.Vars["dt"] = insyra.NewDataTable(insyra.NewDataList(1.0))

	root.SetArgs([]string{"addrow", "dt", "-2.5"})
	if err := root.Execute(); err != nil {
		t.Fatalf("addrow with a negative literal failed: %v (output: %s)", err, out.String())
	}
	dt := ctx.Vars["dt"].(*insyra.DataTable)
	if rows := dt.NumRows(); rows != 2 {
		t.Fatalf("rows = %d want 2", rows)
	}
	if got := dt.GetElement(1, "A"); got != -2.5 {
		t.Errorf("new cell = %#v want -2.5", got)
	}
}

func TestOneShot_AddColAcceptsNegativeLiterals(t *testing.T) {
	root, ctx, out := newOneShotRoot(t)
	ctx.Vars["dt"] = insyra.NewDataTable(insyra.NewDataList(1.0, 2.0))

	root.SetArgs([]string{"addcol", "dt", "-1.5", "-2.5"})
	if err := root.Execute(); err != nil {
		t.Fatalf("addcol with a negative literal failed: %v (output: %s)", err, out.String())
	}
	dt := ctx.Vars["dt"].(*insyra.DataTable)
	if _, cols := dt.Size(); cols != 2 {
		t.Fatalf("cols = %d want 2", cols)
	}
	if got := dt.GetElement(0, "B"); got != -1.5 {
		t.Errorf("new cell = %#v want -1.5", got)
	}
}

func TestOneShot_HelpForNewDLStillRenders(t *testing.T) {
	_, output, err := runOneShot(t, "help", "newdl")
	if err != nil {
		t.Fatalf("help newdl failed: %v", err)
	}
	if !strings.Contains(output, "usage: newdl") {
		t.Errorf("help output = %q, want the newdl usage line", output)
	}
	if strings.Contains(strings.ToLower(output), "unknown flag") {
		t.Errorf("help output = %q, should not report an unknown flag", output)
	}
}

func TestNewDLAddRowAddColDisableFlagParsing(t *testing.T) {
	for _, name := range []string{"newdl", "addrow", "addcol"} {
		handler, ok := Registry[name]
		if !ok {
			t.Fatalf("%s is not registered", name)
		}
		if !handler.DisableFlagParsing {
			t.Errorf("%s should set DisableFlagParsing so negative literals survive", name)
		}
	}
}
