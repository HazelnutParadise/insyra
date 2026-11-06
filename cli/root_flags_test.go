package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/cli/env"
)

// CLI-3: root flags placed before a DisableFlagParsing subcommand still
// apply, instead of being swallowed as data.
func TestRootFlagsBeforeRawArgCommand(t *testing.T) {
	base := t.TempDir()
	env.Default().SetBasePath(base)
	t.Cleanup(func() { env.Default().SetBasePath("") })

	if err := env.Default().Create("e2"); err != nil {
		t.Fatal(err)
	}
	cmd := NewRootCommand()
	cmd.SetArgs([]string{"--env", "e2", "newdl", "1", "2", "3", "as", "ex"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	vars, err := env.Default().RestoreVariables("e2")
	if err != nil {
		t.Fatalf("environment e2 was not written: %v", err)
	}
	dl, ok := vars["ex"].(*insyra.DataList)
	if !ok {
		t.Fatalf("ex not stored in e2: %v", vars)
	}
	if dl.Len() != 3 {
		t.Fatalf("ex has %d items, want 3 (flags leaked into data: %v)", dl.Len(), dl.Data())
	}
	if defVars, err := env.Default().RestoreVariables("default"); err == nil {
		if _, leaked := defVars["ex"]; leaked {
			t.Fatal("ex was written to the default environment")
		}
	}
}

// One-shot `env delete default --force` reaches the command with its flag.
func TestOneShotEnvDeleteDefaultPassesForce(t *testing.T) {
	base := t.TempDir()
	env.Default().SetBasePath(base)
	t.Cleanup(func() { env.Default().SetBasePath("") })
	if err := env.Default().EnsureDefaultEnvironment(); err != nil {
		t.Fatal(err)
	}
	if err := env.Default().Create("work"); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCommand()
	cmd.SetArgs([]string{"--env", "work", "env", "delete", "default"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("env delete default without --force succeeded")
	}
	if !env.Default().Exists("default") {
		t.Fatal("default was deleted without --force")
	}

	cmd = NewRootCommand()
	cmd.SetArgs([]string{"--env", "work", "env", "delete", "default", "--force"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("env delete default --force: %v", err)
	}
	if env.Default().Exists("default") {
		t.Fatal("default is still there after --force")
	}
}

// A raw-argument command run with --env opens only that environment, so an
// unreadable default environment does not stop it.
func TestRawArgCommandIgnoresABrokenDefaultEnvironment(t *testing.T) {
	base := t.TempDir()
	env.Default().SetBasePath(base)
	t.Cleanup(func() { env.Default().SetBasePath("") })
	if err := env.Default().EnsureDefaultEnvironment(); err != nil {
		t.Fatal(err)
	}
	if err := env.Default().Create("good"); err != nil {
		t.Fatal(err)
	}
	defaultPath, err := env.Default().ResolveEnvPath("default")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(defaultPath, "state.json"), []byte(`{"variables":{`), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := NewRootCommand()
	cmd.SetArgs([]string{"--env", "good", "newdl", "7", "as", "z"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("--env good newdl failed because of the default environment: %v", err)
	}
	vars, err := env.Default().RestoreVariables("good")
	if err != nil || vars["z"] == nil {
		t.Fatalf("z not stored in good: %v, %v", vars, err)
	}
	cmd = NewRootCommand()
	cmd.SetArgs([]string{"newdl", "7", "as", "z"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("newdl on the broken default environment succeeded")
	}
}
