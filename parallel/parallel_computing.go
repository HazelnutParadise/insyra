package parallel

import (
	"errors"
	"fmt"
	"reflect"
	"runtime/debug"
	"sync"
)

// ParallelGroup is a list of functions to run at the same time. It can be run
// any number of times, and every Run is independent of the others.
type ParallelGroup struct {
	fns []any
}

// RunningGroup is one run of a ParallelGroup. Wait for it with AwaitResult or
// AwaitNoResult, as many times as you like.
type RunningGroup struct {
	wg      sync.WaitGroup
	results [][]any
	errs    []error // one per function, nil when the function finished
}

// WorkerError reports a function in a group that did not run to completion:
// it panicked, or the value passed to GroupUp could not be called.
type WorkerError struct {
	// Index is the function's position among GroupUp's arguments.
	Index int
	// Panic is the value the function panicked with, or nil.
	Panic any
	// Stack is the goroutine's stack at the panic, or nil.
	Stack []byte
	// Err is why the value could not be called, or nil.
	Err error
}

func (e *WorkerError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("parallel: function %d cannot be called: %v", e.Index, e.Err)
	}
	return fmt.Sprintf("parallel: function %d panicked: %v", e.Index, e.Panic)
}

// Unwrap returns Err, or the panic value when it is an error, so errors.Is and
// errors.As reach the cause.
func (e *WorkerError) Unwrap() error {
	if e.Err != nil {
		return e.Err
	}
	if err, ok := e.Panic.(error); ok {
		return err
	}
	return nil
}

// GroupUp collects zero-argument functions, with any return types, to run at
// the same time. The functions are copied, so changing the slice they came
// from afterwards does not change the group.
func GroupUp(fns ...any) *ParallelGroup {
	return &ParallelGroup{fns: append([]any(nil), fns...)}
}

// Run starts every function in its own goroutine and returns this run. Each
// call starts a new run with its own results. When runs overlap, the same
// function executes in several of them at once, so it must not share
// unsynchronised state with itself.
func (pg *ParallelGroup) Run() *RunningGroup {
	r := &RunningGroup{
		results: make([][]any, len(pg.fns)),
		errs:    make([]error, len(pg.fns)),
	}
	for i, fn := range pg.fns {
		r.wg.Add(1)
		go r.call(i, fn)
	}
	return r
}

// call runs one function and stores what it returned, or why it did not
// finish. A panic stays inside the worker.
func (r *RunningGroup) call(i int, fn any) {
	defer r.wg.Done()

	fnValue := reflect.ValueOf(fn)
	if err := callable(fnValue, fn); err != nil {
		r.errs[i] = &WorkerError{Index: i, Err: err}
		return
	}
	defer func() {
		if p := recover(); p != nil {
			r.errs[i] = &WorkerError{Index: i, Panic: p, Stack: debug.Stack()}
		}
	}()

	out := fnValue.Call(nil)
	if len(out) == 0 {
		return
	}
	values := make([]any, len(out))
	for j, v := range out {
		values[j] = v.Interface()
	}
	r.results[i] = values
}

// callable reports why fn cannot be called with no arguments, or nil.
func callable(v reflect.Value, fn any) error {
	switch {
	case !v.IsValid():
		return errors.New("the value is nil")
	case v.Kind() != reflect.Func:
		return fmt.Errorf("a %T is not a function", fn)
	case v.IsNil():
		return errors.New("the function is nil")
	case v.Type().NumIn() != 0:
		return fmt.Errorf("a %T takes arguments; wrap the call in a closure that takes none", fn)
	}
	return nil
}

// AwaitResult waits for every function and returns what each one returned, in
// GroupUp's order. A slot is nil for a function with no return values and for
// one that failed. The error joins a *WorkerError for every function that
// panicked or could not be called, in order, and is nil when all of them
// finished. An error a function returns itself is part of its results.
func (r *RunningGroup) AwaitResult() ([][]any, error) {
	r.wg.Wait()
	return r.results, errors.Join(r.errs...)
}

// AwaitNoResult waits for every function and returns the same error as
// AwaitResult, without the return values.
func (r *RunningGroup) AwaitNoResult() error {
	_, err := r.AwaitResult()
	return err
}
