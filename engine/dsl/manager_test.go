package dsl

import (
	"os"
	"path/filepath"
	"testing"
)

// A program using engine/dsl gets everything it needs to make a session and
// manage its environments from engine/dsl alone; this file imports nothing
// from cli/.
func TestNewManagerAndSessionWithoutTheCLI(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".idensyra")
	mgr := NewManager(root, "insights")

	if _, err := NewSession(mgr, "analysis", nil); err == nil {
		t.Fatal("NewSession opened an environment that does not exist")
	}
	if err := mgr.Create("analysis"); err != nil {
		t.Fatal(err)
	}
	session, err := NewSession(mgr, "analysis", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Execute("newdl 1 2 3 as x"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "insights", "analysis", "state.json")); err != nil {
		t.Fatalf("state.json is not under the Manager's root: %v", err)
	}

	var infos []EnvironmentInfo
	infos, err = mgr.List()
	if err != nil {
		t.Fatal(err)
	}
	var analysis EnvironmentInfo
	for _, info := range infos {
		if info.Name == "analysis" {
			analysis = info
		}
	}
	if analysis.VariableCount != 1 {
		t.Errorf("List reports %+v for analysis, want one variable", analysis)
	}

	var state *State
	state, err = mgr.LoadState("analysis")
	if err != nil {
		t.Fatal(err)
	}
	isSerialized := func(v SerializedVariable) bool { return v.Type != "" }
	if !isSerialized(state.Variables["x"]) {
		t.Errorf("x is missing from the saved state: %+v", state.Variables)
	}
	var unsaved []UnsavedVariable
	unsaved, err = mgr.SaveVariables("analysis", map[string]any{"f": func() {}})
	if err != nil || len(unsaved) != 1 {
		t.Errorf("SaveVariables of a function: unsaved %v, err %v", unsaved, err)
	}
	var cfg GlobalConfig
	if cfg, err = mgr.LoadGlobalConfig(); err != nil {
		t.Fatalf("LoadGlobalConfig: %v (%+v)", err, cfg)
	}
}

// DefaultManager stores environments where the insyra command does, and each
// call returns a Manager of its own, so moving one does not move another.
func TestDefaultManagerIsTheDefaultLocationAndNotShared(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	a, b := DefaultManager(), DefaultManager()
	if a == b {
		t.Fatal("DefaultManager returned the same Manager twice")
	}
	got, err := a.EnvsPath()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(home, ".insyra", "envs"); got != want {
		t.Errorf("EnvsPath = %q, want %q", got, want)
	}

	a.SetBasePath(filepath.Join(home, "elsewhere"))
	got, err = b.EnvsPath()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(home, ".insyra", "envs"); got != want {
		t.Errorf("moving one DefaultManager moved another: EnvsPath = %q", got)
	}
	got, err = DefaultManager().EnvsPath()
	if err != nil || got != filepath.Join(home, ".insyra", "envs") {
		t.Errorf("a later DefaultManager is at %q, %v", got, err)
	}
}
