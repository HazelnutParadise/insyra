package commands

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strconv"
	"testing"

	"github.com/spf13/cobra"
)

// recordingRoot builds a one-shot root over handlers, each of whose Run is
// replaced by one that records the arguments it receives.
func recordingRoot(t *testing.T, handlers ...*CommandHandler) (*cobra.Command, map[string][]string) {
	t.Helper()
	got := map[string][]string{}
	oldRegistry := Registry
	Registry = map[string]*CommandHandler{}
	t.Cleanup(func() { Registry = oldRegistry })
	for _, h := range handlers {
		copied := *h
		name := copied.Name
		copied.Run = func(ctx *ExecContext, args []string) error {
			got[name] = append([]string{}, args...)
			return nil
		}
		Registry[name] = &copied
	}
	out := &bytes.Buffer{}
	root := &cobra.Command{Use: "insyra", SilenceUsage: true, SilenceErrors: true}
	root.SetOut(out)
	root.SetErr(out)
	for _, sub := range BuildCobraCommands(newTestExecContext(t)) {
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

// BuildCobraCommands builds every command the same way; it names no command.
func TestBuildCobraCommandsNamesNoCommand(t *testing.T) {
	f, err := parser.ParseFile(token.NewFileSet(), "registry.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "BuildCobraCommands" {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			if s, _ := strconv.Unquote(lit.Value); s == "env" || s == "accel" {
				t.Errorf("BuildCobraCommands names the command %q", s)
			}
			return true
		})
		return
	}
	t.Fatal("BuildCobraCommands not found in registry.go")
}

// A flag the shell cannot register would panic in BuildCobraCommands, so
// Register refuses it up front.
func TestRegisterRefusesAFlagTheShellCannotTake(t *testing.T) {
	oldRegistry := Registry
	Registry = map[string]*CommandHandler{}
	t.Cleanup(func() { Registry = oldRegistry })
	run := func(ctx *ExecContext, args []string) error { return nil }
	for _, flags := range [][]CommandFlag{
		{{Name: ""}},
		{{Name: "force"}, {Name: "force", TakesValue: true}},
	} {
		if err := Register(&CommandHandler{Name: "bad", Run: run, Flags: flags}); err == nil {
			t.Errorf("Register accepted flags %+v", flags)
		}
	}
	if _, ok := Registry["bad"]; ok {
		t.Error("a refused command was registered")
	}
}
