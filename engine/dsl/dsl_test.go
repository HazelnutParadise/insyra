package dsl

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/HazelnutParadise/insyra/cli/env"
)

func setupTempHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	t.Setenv("HOMEDRIVE", "")
	t.Setenv("HOMEPATH", "")
	t.Cleanup(func() { env.SetBasePath("") })
	return home
}

func TestNewSessionAndExecute(t *testing.T) {
	setupTempHome(t)

	session, err := NewSession(env.Default(), "", nil)
	if err != nil {
		t.Fatalf("new session failed: %v", err)
	}
	if session == nil {
		t.Fatalf("session should not be nil")
	}

	if err := session.Execute("newdl 1 2 3 as x"); err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if err := session.Execute("mean x"); err != nil {
		t.Fatalf("execute failed: %v", err)
	}
}

func TestNewSessionWithCustomManager(t *testing.T) {
	setupTempHome(t)

	workspace := filepath.Join(t.TempDir(), "ws", ".idensyra")
	mgr := env.NewManager(workspace, "")

	session, err := NewSession(mgr, "default", nil)
	if err != nil {
		t.Fatalf("NewSession with custom manager failed: %v", err)
	}

	want := filepath.Join(workspace, "envs", "default")
	if got := session.Context().EnvPath; got != want {
		t.Fatalf("EnvPath = %q, want %q", got, want)
	}

	if err := session.Execute("newdl 10 20 as v"); err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(want, "state.json")); err != nil {
		t.Fatalf("state.json missing under custom manager root: %v", err)
	}
}

func TestNewSessionRequiresManager(t *testing.T) {
	if _, err := NewSession(nil, "default", nil); err == nil {
		t.Fatalf("expected error when manager is nil")
	}
}
