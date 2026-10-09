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

// Deleting an environment that is a link removes only the link, so a link to
// default is not default; but deleting default while the session works
// through a link to it would pull the session's directory away.
func TestEnvDeleteThroughALink(t *testing.T) {
	ctx := newEnvDeleteContext(t)
	envs, err := ctx.Env.EnvsPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(envs, "default"), filepath.Join(envs, "lnk")); err != nil {
		t.Skipf("cannot make a symbolic link here: %v", err)
	}
	if err := Dispatch(ctx, "env", []string{"delete", "lnk"}); err != nil {
		t.Fatalf("env delete lnk = %v, want it deleted without --force", err)
	}
	if _, err := os.Lstat(filepath.Join(envs, "lnk")); !os.IsNotExist(err) {
		t.Errorf("the link is still there: %v", err)
	}
	if !ctx.Env.Exists("default") {
		t.Fatal("deleting the link deleted default")
	}

	if err := os.Symlink(filepath.Join(envs, "work"), filepath.Join(envs, "via")); err != nil {
		t.Fatal(err)
	}
	ctx.EnvName = "via"
	err = Dispatch(ctx, "env", []string{"delete", "work", "--force"})
	if err == nil || !strings.Contains(err.Error(), "current environment") {
		t.Errorf("env delete work while in a link to it = %v, want the current-environment refusal", err)
	}
	if !ctx.Env.Exists("work") {
		t.Fatal("the session's directory was deleted")
	}
}

// clear, import and rename decided whether they had touched the session's own
// environment by its spelling, so on a file system that ignores case
// `env clear WORK` cleared the file while the session kept the variables and
// wrote them back on the next save.
func TestEnvCommandsOnTheCurrentEnvironmentInAnotherCase(t *testing.T) {
	base := t.TempDir()
	if !caseInsensitiveDir(t, base) {
		t.Skip("the file system distinguishes case, so WORK is another environment")
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
	ctx.Vars["x"] = 1

	if err := Dispatch(ctx, "env", []string{"clear", "WORK"}); err != nil {
		t.Fatalf("env clear WORK: %v", err)
	}
	if _, ok := ctx.Vars["x"]; ok {
		t.Error("env clear WORK left the session's variables, so the next save writes them back")
	}

	export := filepath.Join(t.TempDir(), "e.json")
	ctx.Vars["y"] = 2
	if err := SaveEnvState(ctx); err != nil {
		t.Fatal(err)
	}
	if err := Dispatch(ctx, "env", []string{"export", "work", export}); err != nil {
		t.Fatal(err)
	}
	ctx.Vars = map[string]any{"stale": 3}
	if err := Dispatch(ctx, "env", []string{"import", export, "WORK", "--force"}); err != nil {
		t.Fatalf("env import into WORK: %v", err)
	}
	if _, ok := ctx.Vars["y"]; !ok {
		t.Errorf("env import into WORK did not reload the session's variables: %v", ctx.Vars)
	}

	if err := Dispatch(ctx, "env", []string{"rename", "WORK", "other"}); err != nil {
		t.Fatalf("env rename WORK other: %v", err)
	}
	if ctx.EnvName != "other" {
		t.Errorf("after renaming the current environment the session is in %q, want other", ctx.EnvName)
	}
}
