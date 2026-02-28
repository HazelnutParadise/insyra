package env

import (
	"encoding/json"
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

	if err := Clear("clearable", false); err != nil {
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

func TestClearEnvironmentKeepHistory(t *testing.T) {
	setupTempHome(t)

	if err := Create("clear-keep-history"); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	if err := SaveState("clear-keep-history", map[string]any{"a": 1}); err != nil {
		t.Fatalf("save state failed: %v", err)
	}
	if err := AppendHistory("clear-keep-history", "show t1"); err != nil {
		t.Fatalf("append history failed: %v", err)
	}

	if err := Clear("clear-keep-history", true); err != nil {
		t.Fatalf("clear with keep-history failed: %v", err)
	}

	state, err := LoadState("clear-keep-history")
	if err != nil {
		t.Fatalf("load state failed after clear: %v", err)
	}
	if len(state.Variables) != 0 {
		t.Fatalf("expected empty variables after clear, got %d", len(state.Variables))
	}

	history, err := ReadHistory("clear-keep-history")
	if err != nil {
		t.Fatalf("read history failed after clear: %v", err)
	}
	if len(history) != 1 || history[0] != "show t1" {
		t.Fatalf("expected history to be preserved, got %v", history)
	}
}

func TestExportEnvironmentIncludesStateAndHistory(t *testing.T) {
	setupTempHome(t)

	if err := Create("exportable"); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	if err := SaveState("exportable", map[string]any{"a": 1, "b": "x"}); err != nil {
		t.Fatalf("save state failed: %v", err)
	}
	if err := AppendHistory("exportable", "newdl 1 2 3 as x"); err != nil {
		t.Fatalf("append history failed: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "exportable-env.json")
	if err := Export("exportable", outputPath); err != nil {
		t.Fatalf("export failed: %v", err)
	}

	bytes, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read export file: %v", err)
	}

	var payload ExportPayload
	if err := json.Unmarshal(bytes, &payload); err != nil {
		t.Fatalf("failed to parse export file: %v", err)
	}

	if payload.Environment != "exportable" {
		t.Fatalf("unexpected environment in payload: %s", payload.Environment)
	}
	if payload.State == nil || len(payload.State.Variables) != 2 {
		t.Fatalf("expected 2 variables in exported state")
	}
	if len(payload.History) != 1 || payload.History[0] != "newdl 1 2 3 as x" {
		t.Fatalf("unexpected exported history: %v", payload.History)
	}
}

func TestImportEnvironmentRestoresStateAndHistory(t *testing.T) {
	setupTempHome(t)

	if err := Create("source-env"); err != nil {
		t.Fatalf("create source failed: %v", err)
	}
	if err := SaveState("source-env", map[string]any{"a": 1, "b": "x"}); err != nil {
		t.Fatalf("save state failed: %v", err)
	}
	if err := AppendHistory("source-env", "mean x"); err != nil {
		t.Fatalf("append history failed: %v", err)
	}

	exportFile := filepath.Join(t.TempDir(), "source-export.json")
	if err := Export("source-env", exportFile); err != nil {
		t.Fatalf("export failed: %v", err)
	}

	importedName, err := Import(exportFile, "", false)
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}
	if importedName != "source-env" {
		t.Fatalf("unexpected imported name: %s", importedName)
	}

	state, err := LoadState("source-env")
	if err != nil {
		t.Fatalf("load state after import failed: %v", err)
	}
	if len(state.Variables) != 2 {
		t.Fatalf("expected 2 variables after import, got %d", len(state.Variables))
	}

	history, err := ReadHistory("source-env")
	if err != nil {
		t.Fatalf("read history after import failed: %v", err)
	}
	if len(history) != 1 || history[0] != "mean x" {
		t.Fatalf("unexpected history after import: %v", history)
	}
}

func TestImportEnvironmentNameOverride(t *testing.T) {
	setupTempHome(t)

	if err := Create("origin"); err != nil {
		t.Fatalf("create origin failed: %v", err)
	}
	if err := SaveState("origin", map[string]any{"k": 99}); err != nil {
		t.Fatalf("save state failed: %v", err)
	}

	exportFile := filepath.Join(t.TempDir(), "origin-export.json")
	if err := Export("origin", exportFile); err != nil {
		t.Fatalf("export failed: %v", err)
	}

	importedName, err := Import(exportFile, "restored", false)
	if err != nil {
		t.Fatalf("import with override failed: %v", err)
	}
	if importedName != "restored" {
		t.Fatalf("expected overridden imported name, got %s", importedName)
	}

	if !Exists("restored") {
		t.Fatalf("overridden target environment should exist")
	}
	state, err := LoadState("restored")
	if err != nil {
		t.Fatalf("load restored state failed: %v", err)
	}
	if len(state.Variables) != 1 {
		t.Fatalf("expected 1 variable in restored env, got %d", len(state.Variables))
	}
}

func TestImportEnvironmentNonEmptyRequiresForce(t *testing.T) {
	setupTempHome(t)

	if err := Create("source"); err != nil {
		t.Fatalf("create source failed: %v", err)
	}
	if err := SaveState("source", map[string]any{"new": 1}); err != nil {
		t.Fatalf("save source state failed: %v", err)
	}

	exportFile := filepath.Join(t.TempDir(), "source-export.json")
	if err := Export("source", exportFile); err != nil {
		t.Fatalf("export source failed: %v", err)
	}

	if err := Create("target"); err != nil {
		t.Fatalf("create target failed: %v", err)
	}
	if err := SaveState("target", map[string]any{"old": 999}); err != nil {
		t.Fatalf("save target state failed: %v", err)
	}

	if _, err := Import(exportFile, "target", false); err == nil {
		t.Fatalf("expected import to fail for non-empty target without force")
	}

	if _, err := Import(exportFile, "target", true); err != nil {
		t.Fatalf("expected forced import to succeed: %v", err)
	}

	state, err := LoadState("target")
	if err != nil {
		t.Fatalf("load target state failed: %v", err)
	}
	if _, exists := state.Variables["new"]; !exists {
		t.Fatalf("expected imported variable 'new' in target")
	}
	if _, exists := state.Variables["old"]; exists {
		t.Fatalf("expected old target variable to be overwritten")
	}
}
