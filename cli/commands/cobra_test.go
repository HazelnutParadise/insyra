package commands

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/engine/dsl"
	"github.com/spf13/cobra"
)

// newTestExecContext returns an ExecContext with empty variables and a buffered
// output.
func newTestExecContext(t *testing.T) *ExecContext {
	t.Helper()
	return &ExecContext{
		Vars:   map[string]any{},
		Output: &bytes.Buffer{},
	}
}

// CLI-4: the one-shot dispatcher path writes the sanitized line.
func TestDispatchHistoryIsSanitized(t *testing.T) {
	base := t.TempDir()
	mgr := dsl.NewManager(base, "envs")
	if err := mgr.EnsureDefaultEnvironment(); err != nil {
		t.Fatal(err)
	}
	ctx := newTestExecContext(t)
	ctx.Env = mgr
	ctx.EnvName = "default"
	cmds := BuildCobraCommands(ctx)
	for _, c := range cmds {
		if c.Name() == "db" {
			_ = c.RunE(c, []string{"connect", "bad", "mysql://alice:S3cretPW@127.0.0.1:1/db"})
		}
	}
	lines, err := mgr.ReadHistory("default")
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(lines, "\n")
	if strings.Contains(joined, "S3cretPW") {
		t.Fatalf("history contains the password: %q", joined)
	}
	if !strings.Contains(joined, "db connect bad") {
		t.Fatalf("history lost the command: %q", joined)
	}
	info, err := os.Stat(filepath.Join(base, "envs", "default", "history.txt"))
	if err != nil {
		t.Fatal(err)
	}
	// Windows has no POSIX permission bits: Go reports a writable file there
	// as 0666 whatever mode it was created with, so the check means nothing.
	if perm := info.Mode().Perm(); runtime.GOOS != "windows" && perm&0o077 != 0 {
		t.Fatalf("history.txt is group/world readable: %o", perm)
	}
}

func TestBuildCobraCommands(t *testing.T) {
	run := func(ctx *ExecContext, args []string) error { return nil }
	handlers := map[string]*CommandHandler{
		"show": {Name: "show", Usage: "show <var>", DisableFlagParsing: true, Run: run},
		"help": {Name: "help", Usage: "help", Run: run},
	}
	commands := buildCobraCommands(&ExecContext{Vars: map[string]any{}}, []string{"help", "show"}, handlers, Dispatch)
	if len(commands) != 2 {
		t.Fatalf("expected 2 cobra commands, got %d", len(commands))
	}

	if commands[0].Name() != "help" || commands[1].Name() != "show" {
		t.Fatalf("commands should be in the order given, got %s then %s", commands[0].Name(), commands[1].Name())
	}
	if !commands[1].DisableFlagParsing {
		t.Fatalf("show command should keep DisableFlagParsing=true")
	}

	names, _ := SnapshotRegistry()
	shell := BuildCobraCommands(newTestExecContext(t))
	if len(shell) != len(names) {
		t.Fatalf("BuildCobraCommands made %d commands for %d registered", len(shell), len(names))
	}
	for i, c := range shell {
		if c.Name() != names[i] {
			t.Fatalf("command %d is %s, want %s: not in name order", i, c.Name(), names[i])
		}
	}
}

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
		Env:     dsl.NewManager(t.TempDir(), ""),
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

// recordingRoot builds a one-shot root over handlers whose commands, instead of
// running, record the arguments they would have received.
func recordingRoot(t *testing.T, handlers ...*CommandHandler) (*cobra.Command, map[string][]string) {
	t.Helper()
	got := map[string][]string{}
	byName := map[string]*CommandHandler{}
	names := []string{}
	for _, h := range handlers {
		byName[h.Name] = h
		names = append(names, h.Name)
	}
	record := func(ctx *ExecContext, name string, args []string) error {
		got[name] = append([]string{}, args...)
		return nil
	}
	out := &bytes.Buffer{}
	root := &cobra.Command{Use: "insyra", SilenceUsage: true, SilenceErrors: true}
	root.SetOut(out)
	root.SetErr(out)
	for _, sub := range buildCobraCommands(newTestExecContext(t), names, byName, record) {
		root.AddCommand(sub)
	}
	return root, got
}

// #261 (CL-3): a command declares the flags its one-shot form takes where it is
// registered, and BuildCobraCommands hands each one that is set to Run.
func TestRegisteredFlagsReachRun(t *testing.T) {
	cases := []struct {
		args []string
		want []string
	}{
		{[]string{"flagged", "import", "f", "--force"}, []string{"import", "f", "--force"}},
		{[]string{"flagged", "IMPORT", "f", "--force"}, []string{"IMPORT", "f", "--force"}},
		{[]string{"flagged", "clear", "--force"}, []string{"clear"}},
		{[]string{"flagged", "x", "--mode", "gpu"}, []string{"x", "--mode", "gpu"}},
		{[]string{"flagged", "x", "--mode", "  "}, []string{"x"}},
		{[]string{"flagged", "import", "f", "--force", "--mode", "cpu"}, []string{"import", "f", "--force", "--mode", "cpu"}},
	}
	for _, tc := range cases {
		root, got := recordingRoot(t, &CommandHandler{
			Name: "flagged",
			Flags: []CommandFlag{
				{Name: "force", Usage: "overwrite", Form: "import"},
				{Name: "mode", Usage: "the mode", TakesValue: true},
			},
		})
		root.SetArgs(tc.args)
		if err := root.Execute(); err != nil {
			t.Fatalf("%v: %v", tc.args, err)
		}
		if !reflect.DeepEqual(got["flagged"], tc.want) {
			t.Errorf("%v: Run got %q, want %q", tc.args, got["flagged"], tc.want)
		}
	}

	root, _ := recordingRoot(t, &CommandHandler{
		Name:  "flagged",
		Flags: []CommandFlag{{Name: "force", Usage: "overwrite"}, {Name: "mode", Usage: "the mode", TakesValue: true}},
	})
	sub, _, err := root.Find([]string{"flagged"})
	if err != nil {
		t.Fatal(err)
	}
	if f := sub.Flags().Lookup("force"); f == nil || f.Usage != "overwrite" || f.Value.Type() != "bool" {
		t.Errorf("--force is %+v, want a bool flag with its usage", f)
	}
	if f := sub.Flags().Lookup("mode"); f == nil || f.Usage != "the mode" || f.Value.Type() != "string" {
		t.Errorf("--mode is %+v, want a string flag with its usage", f)
	}
}

// The one-shot flags of env and accel reach Run exactly as they did when
// BuildCobraCommands named the two commands itself.
func TestEnvAndAccelOneShotFlagsAsBefore(t *testing.T) {
	envHandler, ok := LookupCommand("env")
	if !ok {
		t.Fatal("env is not registered")
	}
	accelHandler, ok := LookupCommand("accel")
	if !ok {
		t.Fatal("accel is not registered")
	}
	cases := []struct {
		args []string
		want []string
	}{
		{[]string{"env", "clear", "a", "--keep-history"}, []string{"clear", "a", "--keep-history"}},
		{[]string{"env", "CLEAR", "a", "--keep-history"}, []string{"CLEAR", "a", "--keep-history"}},
		{[]string{"env", "clear", "a"}, []string{"clear", "a"}},
		{[]string{"env", "import", "f.json", "t", "--force"}, []string{"import", "f.json", "t", "--force"}},
		{[]string{"env", "clear", "a", "--force"}, []string{"clear", "a"}},
		{[]string{"env", "import", "f.json", "--keep-history"}, []string{"import", "f.json"}},
		{[]string{"accel", "devices", "--mode", "cpu"}, []string{"devices", "--mode", "cpu"}},
		{[]string{"accel", "devices"}, []string{"devices"}},
	}
	for _, tc := range cases {
		root, got := recordingRoot(t, envHandler, accelHandler)
		root.SetArgs(tc.args)
		if err := root.Execute(); err != nil {
			t.Fatalf("%v: %v", tc.args, err)
		}
		if !reflect.DeepEqual(got[tc.args[0]], tc.want) {
			t.Errorf("%v: Run got %q, want %q", tc.args, got[tc.args[0]], tc.want)
		}
	}

	root, _ := recordingRoot(t, envHandler, accelHandler)
	usages := map[[2]string]string{
		{"env", "keep-history"}: "With 'env clear', keep command history",
		{"env", "force"}:        "With 'env import', overwrite non-empty target environment",
		{"accel", "mode"}:       "Acceleration mode: auto|cpu|gpu|strict-gpu",
	}
	for key, usage := range usages {
		sub, _, err := root.Find([]string{key[0]})
		if err != nil {
			t.Fatal(err)
		}
		if f := sub.Flags().Lookup(key[1]); f == nil || f.Usage != usage {
			t.Errorf("%s --%s is %+v, want usage %q", key[0], key[1], f, usage)
		}
	}
}

// BuildCobraCommands builds every command the same way; it names no command
// to add or forward a flag.
func TestBuildCobraCommandsNamesNoCommand(t *testing.T) {
	f, err := parser.ParseFile(token.NewFileSet(), "cobra.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		found++
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			if s, _ := strconv.Unquote(lit.Value); s == "env" || s == "accel" {
				t.Errorf("%s names the command %q", fn.Name.Name, s)
			}
			return true
		})
	}
	if found == 0 {
		t.Fatal("no functions found in cobra.go")
	}
}
