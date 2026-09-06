package commands

import (
	"fmt"
	"sync"
	"testing"
)

func TestRegistryConcurrentRegisterAndDispatch(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			name := fmt.Sprintf("zz-batch3-%d", i)
			if err := Register(&CommandHandler{Name: name, Run: func(*ExecContext, []string) error { return nil }}); err != nil {
				t.Errorf("register %s: %v", name, err)
				return
			}
			if err := Dispatch(&ExecContext{}, name, nil); err != nil {
				t.Errorf("dispatch %s: %v", name, err)
			}
		}(i)
	}
	wg.Wait()
}
