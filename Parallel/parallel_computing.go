package parallel

import (
	"reflect"
	"sync"
)

type ParallelGroup struct {
	fns     []interface{}
	results [][]interface{}
	wg      sync.WaitGroup
}

// GroupUp initializes a new ParallelGroup with the given functions.
func GroupUp(fns ...interface{}) *ParallelGroup {
	return &ParallelGroup{
		fns:     fns,
		results: make([][]interface{}, len(fns)),
	}
}

// Run starts the execution of all functions in parallel.
func (pg *ParallelGroup) Run() *ParallelGroup {
	for i, fn := range pg.fns {
		pg.wg.Add(1)
		go func(i int, fn interface{}) {
			defer pg.wg.Done()
			fnValue := reflect.ValueOf(fn)
			resultValues := fnValue.Call(nil)
			results := make([]interface{}, len(resultValues))
			for j, v := range resultValues {
				results[j] = v.Interface()
			}
			pg.results[i] = results
		}(i, fn)
	}
	return pg
}

// AwaitResult waits for all functions to complete and returns their results.
func (pg *ParallelGroup) AwaitResult() [][]interface{} {
	pg.wg.Wait()
	return pg.results
}
