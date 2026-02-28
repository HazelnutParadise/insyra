package env

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTempHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	t.Setenv("HOMEDRIVE", "")
	t.Setenv("HOMEPATH", "")
	return home
}

func TestEnvironmentCRUD(t *testing.T) {
	home := setupTempHome(t)

	if err := EnsureDefaultEnvironment(); err != nil {
		t.Fatalf("ensure default env failed: %v", err)
	}
	if !Exists("default") {
		t.Fatalf("default environment should exist")
	}

	if err := Create("test-env"); err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if !Exists("test-env") {
		t.Fatalf("test-env should exist after create")
	}

	if err := Rename("test-env", "renamed-env"); err != nil {
		t.Fatalf("rename failed: %v", err)
	}
	if Exists("test-env") {
		t.Fatalf("old environment name should not exist after rename")
	}
	if !Exists("renamed-env") {
		t.Fatalf("renamed environment should exist")
	}

	openedPath, err := Open("renamed-env")
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}
	expected := filepath.Join(home, ".insyra", "envs", "renamed-env")
	if openedPath != expected {
		t.Fatalf("unexpected open path: got %s want %s", openedPath, expected)
	}

	if err := Delete("renamed-env"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if Exists("renamed-env") {
		t.Fatalf("environment should be deleted")
	}
}

func TestStateAndHistoryPersistence(t *testing.T) {
	setupTempHome(t)

	if err := Create("persist"); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	vars := map[string]any{"x": 123, "name": "demo"}
	if err := SaveState("persist", vars); err != nil {
		t.Fatalf("save state failed: %v", err)
	}
	state, err := LoadState("persist")
	if err != nil {
		t.Fatalf("load state failed: %v", err)
	}
	if len(state.Variables) != 2 {
		t.Fatalf("expected 2 variables, got %d", len(state.Variables))
	}

	if err := AppendHistory("persist", "show t1"); err != nil {
		t.Fatalf("append history failed: %v", err)
	}
	if err := AppendHistory("persist", "summary t1"); err != nil {
		t.Fatalf("append history failed: %v", err)
	}
	entries, err := ReadHistory("persist")
	if err != nil {
		t.Fatalf("read history failed: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 history entries, got %d", len(entries))
	}
	if entries[0] != "show t1" || entries[1] != "summary t1" {
		t.Fatalf("unexpected history contents: %v", entries)
	}

	envPath, err := ResolveEnvPath("persist")
	if err != nil {
		t.Fatalf("resolve path failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(envPath, "state.json")); err != nil {
		t.Fatalf("state.json missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(envPath, "history.txt")); err != nil {
		t.Fatalf("history.txt missing: %v", err)
	}
}

func TestClearEnvironmentStateAndHistory(t *testing.T) {
	setupTempHome(t)

	if err := Create("clearable"); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	if err := SaveState("clearable", map[string]any{"a": 1, "b": "x"}); err != nil {
		t.Fatalf("save state failed: %v", err)
	}
	if err := AppendHistory("clearable", "show t1"); err != nil {
		t.Fatalf("append history failed: %v", err)
	}

	if err := Clear("clearable"); err != nil {
		t.Fatalf("clear failed: %v", err)
	}

	state, err := LoadState("clearable")
	if err != nil {
		t.Fatalf("load state failed after clear: %v", err)
	}
	if len(state.Variables) != 0 {
		t.Fatalf("expected empty variables after clear, got %d", len(state.Variables))
	}

	history, err := ReadHistory("clearable")
	if err != nil {
		t.Fatalf("read history failed after clear: %v", err)
	}
	if len(history) != 0 {
		t.Fatalf("expected empty history after clear, got %d entries", len(history))
	}

	envPath, err := ResolveEnvPath("clearable")
	if err != nil {
		t.Fatalf("resolve env path failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(envPath, "config.json")); err != nil {
		t.Fatalf("config.json should be kept after clear: %v", err)
	}
}
