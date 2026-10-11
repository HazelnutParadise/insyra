package dsl

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestNewManagerCreatesAndManagesSessionEnvironment(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".idensyra")
	mgr := NewManager(root, "workspaces")
	session, err := NewSession(mgr, "analysis", nil)
	if err != nil {
		t.Fatal(err)
	}
	if session == nil {
		t.Fatal("NewSession returned nil")
	}
	if !mgr.Exists("analysis") {
		t.Fatal("NewSession did not create the named environment")
	}
	if _, err := os.Stat(filepath.Join(root, "workspaces", "analysis", "state.json")); err != nil {
		t.Fatalf("state file was not created under manager root: %v", err)
	}
}

func TestConcurrentNewSessionsCreateOneEnvironmentSafely(t *testing.T) {
	mgr := NewManager(t.TempDir(), "")
	const count = 12
	var wg sync.WaitGroup
	errs := make(chan error, count)
	for range count {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := NewSession(mgr, "shared", nil)
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Errorf("NewSession: %v", err)
		}
	}
	if !mgr.Exists("shared") {
		t.Fatal("concurrent NewSession calls did not create the environment")
	}
}
