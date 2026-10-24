package env

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// Two processes on one environment each restore, add a variable and save. The
// later save must not erase the variable the earlier one wrote.
func TestSavesFromTwoProcessesKeepBothProcessesVariables(t *testing.T) {
	root := t.TempDir()
	setup := NewManager(root, "")
	if err := setup.EnsureDefaultEnvironment(); err != nil {
		t.Fatal(err)
	}
	if err := setup.SaveState("default", map[string]any{"shared": 1, "gone": 2}); err != nil {
		t.Fatal(err)
	}

	a, b := NewManager(root, ""), NewManager(root, "")
	varsA, err := a.RestoreVariables("default")
	if err != nil {
		t.Fatal(err)
	}
	varsB, err := b.RestoreVariables("default")
	if err != nil {
		t.Fatal(err)
	}
	varsA["from_a"] = 10
	delete(varsA, "gone")
	varsB["from_b"] = 20
	varsB["shared"] = 5

	if err := a.SaveState("default", varsA); err != nil {
		t.Fatal(err)
	}
	if err := b.SaveState("default", varsB); err != nil {
		t.Fatal(err)
	}

	got, err := NewManager(root, "").RestoreVariables("default")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"from_a", "from_b", "shared"} {
		if _, ok := got[name]; !ok {
			t.Errorf("variable %q lost; have %v", name, got)
		}
	}
	if _, ok := got["gone"]; ok {
		t.Errorf("variable removed by one process came back: %v", got)
	}
	if got["shared"] != int(5) && got["shared"] != int64(5) && got["shared"] != 5.0 {
		t.Errorf("shared = %#v, want the value B set", got["shared"])
	}
}

func TestConcurrentSavesNeverLoseAVariable(t *testing.T) {
	root := t.TempDir()
	if err := NewManager(root, "").EnsureDefaultEnvironment(); err != nil {
		t.Fatal(err)
	}
	const n = 12
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m := NewManager(root, "")
			vars, err := m.RestoreVariables("default")
			if err != nil {
				t.Errorf("restore: %v", err)
				return
			}
			vars[fmt.Sprintf("v%d", i)] = i
			if err := m.SaveState("default", vars); err != nil {
				t.Errorf("save: %v", err)
			}
		}()
	}
	wg.Wait()
	// Each goroutine restored before or after the others saved, but every save
	// merges over the file, so all n names must be present at the end.
	got, err := NewManager(root, "").RestoreVariables("default")
	if err != nil {
		t.Fatal(err)
	}
	for i := range n {
		if _, ok := got[fmt.Sprintf("v%d", i)]; !ok {
			t.Errorf("v%d lost; have %d variables", i, len(got))
		}
	}
	matches, _ := filepath.Glob(filepath.Join(root, "envs", "default", "state.json.*.tmp"))
	if len(matches) != 0 || fileExists(filepath.Join(root, "envs", "default", stateLockName)) {
		t.Errorf("temporary or lock file left behind: %v", matches)
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
