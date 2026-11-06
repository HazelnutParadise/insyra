package env

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// A session opening an environment another one is creating must find a whole
// state.json, or the stop on an unreadable file refuses it.
func TestOpeningAnEnvironmentWhileItIsCreatedReadsAWholeState(t *testing.T) {
	m := NewManager(t.TempDir(), "")
	if err := m.EnsureBaseStructure(); err != nil {
		t.Fatal(err)
	}
	for round := range 200 {
		name := fmt.Sprintf("shared%d", round)
		var wg sync.WaitGroup
		errs := make(chan error, 1)
		wg.Add(2)
		go func() {
			defer wg.Done()
			if err := m.Create(name); err != nil && !errors.Is(err, ErrAlreadyExists) {
				errs <- err
			}
		}()
		go func() {
			defer wg.Done()
			path, err := m.ResolveEnvPath(name)
			if err != nil {
				errs <- err
				return
			}
			// Open the moment state.json appears, which is when a
			// part-written file would be seen.
			deadline := time.Now().Add(5 * time.Second)
			for {
				if _, err := os.Stat(filepath.Join(path, "state.json")); err == nil {
					break
				}
				if time.Now().After(deadline) {
					errs <- errors.New("state.json never appeared")
					return
				}
			}
			if _, err := m.OpenVariables(name); err != nil {
				select {
				case errs <- err:
				default:
				}
			}
		}()
		wg.Wait()
		select {
		case err := <-errs:
			t.Fatalf("round %d: %v", round, err)
		default:
		}
	}
}
