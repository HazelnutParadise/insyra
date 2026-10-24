package env

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenVariablesRefusesAnUnreadableStateAndLeavesItAlone(t *testing.T) {
	root := t.TempDir()
	m := NewManager(root, "")
	if err := m.EnsureDefaultEnvironment(); err != nil {
		t.Fatal(err)
	}
	if err := m.SaveState("default", map[string]any{"x": 1}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "envs", "default", "state.json")
	full, _ := os.ReadFile(path)
	broken := full[:len(full)/2]
	if err := os.WriteFile(path, broken, 0o644); err != nil {
		t.Fatal(err)
	}

	vars, err := m.OpenVariables("default")
	if err == nil || vars != nil {
		t.Fatalf("got (%v, %v), want an error and no variables", vars, err)
	}
	if !strings.Contains(err.Error(), "state.json") {
		t.Errorf("the error does not name the file: %v", err)
	}
	if after, _ := os.ReadFile(path); string(after) != string(broken) {
		t.Error("the unreadable file was changed")
	}
}

func TestOpenVariablesStartsEmptyWithoutAStateFile(t *testing.T) {
	root := t.TempDir()
	m := NewManager(root, "")
	if err := m.EnsureDefaultEnvironment(); err != nil {
		t.Fatal(err)
	}
	_ = os.Remove(filepath.Join(root, "envs", "default", "state.json"))
	vars, err := m.OpenVariables("default")
	if err != nil || vars == nil || len(vars) != 0 {
		t.Fatalf("got (%v, %v), want an empty map and no error", vars, err)
	}
}
