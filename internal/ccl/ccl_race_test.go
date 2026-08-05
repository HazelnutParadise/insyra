package ccl

import (
	"sync"
	"testing"
)

func TestEvaluate_ConcurrentExpressionsHaveIndependentDepth(t *testing.T) {
	node, err := CompileExpression("SQRT(A*A + B*B) * 2 + A / B - 1")
	if err != nil {
		t.Fatal(err)
	}

	const workers = 8
	const evaluationsPerWorker = 100
	errs := make(chan error, workers)
	var wg sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			ctx, err := NewMapContext(map[string][]any{
				"A": {float64(worker + 1)},
				"B": {float64(worker + 2)},
			})
			if err != nil {
				errs <- err
				return
			}
			for i := 0; i < evaluationsPerWorker; i++ {
				if _, err := Evaluate(node, ctx); err != nil {
					errs <- err
					return
				}
			}
		}(worker)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
}
