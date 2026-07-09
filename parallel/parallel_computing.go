package parallel

import (
	"fmt"
	"reflect"
	"sync"
)

type ParallelGroup struct {
	fns     []any
	results [][]any
	wg      sync.WaitGroup
}

// GroupUp initializes a new ParallelGroup with the given functions.
func GroupUp(fns ...any) *ParallelGroup {
	return &ParallelGroup{
		fns:     fns,
		results: make([][]any, len(fns)),
	}
}

// Run starts the execution of all functions in parallel goroutines.
func (pg *ParallelGroup) Run() *ParallelGroup {
	for i, fn := range pg.fns {
		pg.wg.Add(1)
		go func(i int, fn any) {
			defer pg.wg.Done()
			// Recover so a panic in one worker is surfaced via its result slot
			// instead of crashing the whole process.
			defer func() {
				if r := recover(); r != nil {
					pg.results[i] = []any{fmt.Errorf("parallel worker %d panicked: %v", i, r)}
				}
			}()
			fnValue := reflect.ValueOf(fn)
			// Validate the value is a zero-argument function before calling;
			// reflect.Call(nil) on a non-func or an arity mismatch is a fatal panic.
			if fnValue.Kind() != reflect.Func {
				pg.results[i] = []any{fmt.Errorf("parallel worker %d: value is not a function (%T)", i, fn)}
				return
			}
			if t := fnValue.Type(); t.NumIn() != 0 || t.IsVariadic() {
				pg.results[i] = []any{fmt.Errorf("parallel worker %d: function must take no arguments", i)}
				return
			}
			resultValues := fnValue.Call(nil)
			if len(resultValues) > 0 {
				results := make([]any, len(resultValues))
				for j, v := range resultValues {
					results[j] = v.Interface()
				}
				pg.results[i] = results
			}
		}(i, fn)
	}
	return pg
}

// AwaitResult waits for all functions to complete and returns their results.
func (pg *ParallelGroup) AwaitResult() [][]any {
	pg.wg.Wait()
	return pg.results
}

// AwaitNoResult waits for all functions to complete without returning results.
// This is optimized for functions that do not return values, avoiding result collection overhead.
func (pg *ParallelGroup) AwaitNoResult() {
	pg.wg.Wait()
}
