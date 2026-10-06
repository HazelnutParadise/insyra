package env

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// Exactly one of several concurrent Creates of the same environment succeeds;
// the rest report that it already exists.
func TestConcurrentCreateHasOneWinner(t *testing.T) {
	for round := 0; round < 50; round++ {
		mgr := NewManager(t.TempDir(), "")
		const n = 8
		var wg sync.WaitGroup
		errs := make(chan error, n)
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				errs <- mgr.Create("shared")
			}()
		}
		wg.Wait()
		close(errs)
		won := 0
		for err := range errs {
			if err == nil {
				won++
			}
		}
		if won != 1 {
			t.Fatalf("round %d: %d Creates succeeded, want 1", round, won)
		}
	}
}

// Create fills in the default files only where none exists, so a Create that
// loses the race to the environment's folder cannot wipe a state.json the
// winner has already saved into.
func TestCreateDoesNotOverwriteAnExistingFile(t *testing.T) {
	mgr := NewManager(t.TempDir(), "")
	path, err := mgr.ResolveEnvPath("kept")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	state := filepath.Join(path, "state.json")
	if err := os.WriteFile(state, []byte(`{"variables":{"x":{"type":"Raw","data":1}},"lastAccess":""}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeDefaultFiles(path); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(state)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `{"variables":{"x":{"type":"Raw","data":1}},"lastAccess":""}` {
		t.Errorf("state.json was overwritten: %s", got)
	}
	for _, name := range []string{"history.txt", "config.json"} {
		if _, err := os.Stat(filepath.Join(path, name)); err != nil {
			t.Errorf("%s was not created: %v", name, err)
		}
	}
}

// EnsureDefaultEnvironment called from several goroutines at once succeeds in
// every one of them.
func TestConcurrentEnsureDefaultEnvironment(t *testing.T) {
	for round := 0; round < 50; round++ {
		mgr := NewManager(t.TempDir(), "")
		var wg sync.WaitGroup
		errs := make(chan error, 4)
		for i := 0; i < 4; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				errs <- mgr.EnsureDefaultEnvironment()
			}()
		}
		wg.Wait()
		close(errs)
		for err := range errs {
			if err != nil {
				t.Fatalf("round %d: %v", round, err)
			}
		}
	}
}
