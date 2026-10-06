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

// A program names its environment in code and does not know on its first run
// whether it exists yet, so NewSession creates a missing one and reuses one
// that is there.
func TestNewSessionCreatesAMissingEnvironmentAndReusesAnExistingOne(t *testing.T) {
	mgr := NewManager(t.TempDir(), "")

	first, err := NewSession(mgr, "analysis", nil)
	if err != nil {
		t.Fatalf("NewSession on a missing environment: %v", err)
	}
	if !mgr.Exists("analysis") {
		t.Fatal("NewSession did not create the environment")
	}
	if err := first.Execute("newdl 1 2 3 as x"); err != nil {
		t.Fatal(err)
	}

	second, err := NewSession(mgr, "analysis", nil)
	if err != nil {
		t.Fatalf("NewSession on an existing environment: %v", err)
	}
	if _, ok := second.Context().Vars["x"]; !ok {
		t.Errorf("the second session lost x: %v", second.Context().Vars)
	}
}

func TestNewSessionRefusesANameItCannotCreate(t *testing.T) {
	mgr := NewManager(t.TempDir(), "")
	for _, bad := range []string{"a/b", "..", "with space"} {
		if _, err := NewSession(mgr, bad, nil); err == nil {
			t.Errorf("NewSession accepted the environment name %q", bad)
		}
	}
	infos, err := mgr.List()
	if err != nil {
		t.Fatal(err)
	}
	for _, info := range infos {
		if info.Name != "default" {
			t.Errorf("a refused name left the environment %q behind", info.Name)
		}
	}
}

// Two sessions opened at once on the same missing environment both succeed:
// whichever creates it, the other uses it.
func TestNewSessionOnAMissingEnvironmentFromTwoGoroutines(t *testing.T) {
	for round := 0; round < 20; round++ {
		mgr := NewManager(t.TempDir(), "")
		errs := make(chan error, 2)
		for i := 0; i < 2; i++ {
			go func() {
				_, err := NewSession(mgr, "shared", nil)
				errs <- err
			}()
		}
		for i := 0; i < 2; i++ {
			if err := <-errs; err != nil {
				t.Fatalf("round %d: %v", round, err)
			}
		}
	}
}
