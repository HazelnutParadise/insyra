package commands

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestRegistryConcurrentRegisterAndDispatch(t *testing.T) {
	// The registry is global and the docs tests compare it with
	// Docs/cli-dsl.md, so every name registered here is unique to this run
	// and removed again, or a second run (-count, -shuffle) sees it.
	prefix := fmt.Sprintf("zz-registry-race-%d", time.Now().UnixNano())
	names := make([]string, 8)
	for i := range names {
		names[i] = fmt.Sprintf("%s-%d", prefix, i)
	}
	t.Cleanup(func() {
		registryMu.Lock()
		defer registryMu.Unlock()
		for _, name := range names {
			delete(Registry, name)
		}
	})

	var wg sync.WaitGroup
	for _, name := range names {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			if err := Register(&CommandHandler{Name: name, Run: func(*ExecContext, []string) error { return nil }}); err != nil {
				t.Errorf("register %s: %v", name, err)
				return
			}
			if err := Dispatch(&ExecContext{}, name, nil); err != nil {
				t.Errorf("dispatch %s: %v", name, err)
			}
		}(name)
	}
	wg.Wait()
}
