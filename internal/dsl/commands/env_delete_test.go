package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra/internal/dsl/env"
)

func newEnvDeleteContext(t *testing.T) *ExecContext {
	t.Helper()
	mgr := env.NewManager(t.TempDir(), "")
	if err := mgr.EnsureDefaultEnvironment(); err != nil {
		t.Fatal(err)
	}
	if err := mgr.Create("work"); err != nil {
		t.Fatal(err)
	}
	ctx := newTestExecContext(t)
	ctx.Env = mgr
	ctx.EnvName = "work"
	return ctx
}

// `env delete default` removed the default environment's variables and
// history without a word (#320).
func TestEnvDeleteDefaultNeedsForce(t *testing.T) {
	ctx := newEnvDeleteContext(t)
	err := Dispatch(ctx, "env", []string{"delete", "default"})
	if err == nil || !strings.Contains(err.Error(), "--force") {
		t.Fatalf("env delete default = %v, want a refusal that mentions --force", err)
	}
	if !ctx.Env.Exists("default") {
		t.Fatal("default was deleted without --force")
	}
	if err := Dispatch(ctx, "env", []string{"delete", "default", "--force"}); err != nil {
		t.Fatalf("env delete default --force: %v", err)
	}
	if ctx.Env.Exists("default") {
		t.Fatal("default is still there after --force")
	}
}

func TestEnvDeleteOtherEnvironmentsNeedNoForce(t *testing.T) {
	ctx := newEnvDeleteContext(t)
	for _, name := range []string{"a", "b"} {
		if err := ctx.Env.Create(name); err != nil {
			t.Fatal(err)
		}
	}
	if err := Dispatch(ctx, "env", []string{"delete", "a"}); err != nil {
		t.Fatalf("env delete a: %v", err)
	}
	if err := Dispatch(ctx, "env", []string{"delete", "--force", "b"}); err != nil {
		t.Fatalf("env delete --force b: %v", err)
	}
	if ctx.Env.Exists("a") || ctx.Env.Exists("b") {
		t.Fatal("an environment was not deleted")
	}
}

func TestEnvDeleteForceDoesNotDeleteTheCurrentEnvironment(t *testing.T) {
	ctx := newEnvDeleteContext(t)
	ctx.EnvName = "default"
	err := Dispatch(ctx, "env", []string{"delete", "default", "--force"})
	if err == nil || !strings.Contains(err.Error(), "current environment") {
		t.Fatalf("deleting the current environment with --force = %v", err)
	}
	if !ctx.Env.Exists("default") {
		t.Fatal("the current environment was deleted")
	}
}

func TestEnvDeleteRefusesAnUnknownFlag(t *testing.T) {
	ctx := newEnvDeleteContext(t)
	if err := ctx.Env.Create("a"); err != nil {
		t.Fatal(err)
	}
	err := Dispatch(ctx, "env", []string{"delete", "a", "--frce"})
	if err == nil || !strings.Contains(err.Error(), "--frce") {
		t.Fatalf("env delete a --frce = %v, want an error naming the flag", err)
	}
	if !ctx.Env.Exists("a") {
		t.Fatal("a was deleted despite the unknown flag")
	}
}

// The help of every command that replaces or removes data says so.
func TestHelpSaysWhatIsReplacedOrRemoved(t *testing.T) {
	cases := map[string][]string{
		"save": {"replaces"},
		"plot": {"<type>.html", "replaces"},
		"env":  {"--force", "history", "replaces"},
	}
	for name, wants := range cases {
		handler, ok := LookupCommand(name)
		if !ok {
			t.Fatalf("%s is not registered", name)
		}
		forms := strings.Join(handler.Forms, "\n")
		for _, want := range wants {
			if !strings.Contains(forms, want) {
				t.Errorf("help %s does not mention %q:\n%s", name, want, forms)
			}
		}
	}
}

// caseInsensitiveDir reports whether dir's file system ignores letter case.
func caseInsensitiveDir(t *testing.T, dir string) bool {
	t.Helper()
	if err := os.Mkdir(filepath.Join(dir, "probe"), 0o700); err != nil {
		t.Fatal(err)
	}
	_, err := os.Stat(filepath.Join(dir, "PROBE"))
	return err == nil
}

// On a file system that ignores case, `Default` is the default environment's
// directory, so the refusals compared by spelling let it through.
func TestEnvDeleteRefusalsHoldForANameInAnotherCase(t *testing.T) {
	base := t.TempDir()
	if !caseInsensitiveDir(t, base) {
		t.Skip("the file system distinguishes case, so Default is another environment")
	}
	mgr := env.NewManager(base, "")
	if err := mgr.EnsureDefaultEnvironment(); err != nil {
		t.Fatal(err)
	}
	if err := mgr.Create("work"); err != nil {
		t.Fatal(err)
	}
	ctx := newTestExecContext(t)
	ctx.Env = mgr
	ctx.EnvName = "work"
	if err := Dispatch(ctx, "env", []string{"delete", "Default"}); err == nil || !strings.Contains(err.Error(), "--force") {
		t.Errorf("env delete Default = %v, want the refusal that mentions --force", err)
	}
	if err := Dispatch(ctx, "env", []string{"delete", "WORK", "--force"}); err == nil || !strings.Contains(err.Error(), "current environment") {
		t.Errorf("env delete WORK while in work = %v, want the current-environment refusal", err)
	}
	if !mgr.Exists("default") || !mgr.Exists("work") {
		t.Fatal("an environment was deleted through another spelling")
	}
}
