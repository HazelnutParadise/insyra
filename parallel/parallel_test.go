package parallel_test

import (
	"errors"
	"reflect"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra/parallel"
)

// workerErrors lists the WorkerErrors joined into err, in order.
func workerErrors(t *testing.T, err error) []*parallel.WorkerError {
	t.Helper()
	if err == nil {
		return nil
	}
	var list []error
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		list = joined.Unwrap()
	} else {
		list = []error{err}
	}
	out := make([]*parallel.WorkerError, 0, len(list))
	for _, e := range list {
		var we *parallel.WorkerError
		if !errors.As(e, &we) {
			t.Fatalf("%v is not a *parallel.WorkerError", e)
		}
		out = append(out, we)
	}
	return out
}

func TestAwaitResultReturnsWhatEachFunctionReturned(t *testing.T) {
	var counter atomic.Int64
	results, err := parallel.GroupUp(
		func() int { time.Sleep(10 * time.Millisecond); return 1 },
		func() (string, float64) { return "hello", 2.5 },
		func() { counter.Add(1) },
	).Run().AwaitResult()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := [][]any{{1}, {"hello", 2.5}, nil}
	if !reflect.DeepEqual(results, want) {
		t.Errorf("results %v, want %v", results, want)
	}
	if counter.Load() != 1 {
		t.Errorf("the function without a return value ran %d times, want 1", counter.Load())
	}
}

func TestAwaitNoResultWaitsForEveryFunction(t *testing.T) {
	var counter atomic.Int64
	start := time.Now()
	err := parallel.GroupUp(
		func() { time.Sleep(50 * time.Millisecond); counter.Add(1) },
		func() { counter.Add(1) },
	).Run().AwaitNoResult()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if elapsed := time.Since(start); elapsed < 40*time.Millisecond {
		t.Errorf("AwaitNoResult returned after %v, before the slow function finished", elapsed)
	}
	if counter.Load() != 2 {
		t.Errorf("counter %d, want 2", counter.Load())
	}
}

func TestAGroupRunsMoreThanOnce(t *testing.T) {
	var calls atomic.Int64
	g := parallel.GroupUp(func() int64 { return calls.Add(1) })

	first := g.Run()
	second := g.Run()
	a, errA := first.AwaitResult()
	b, errB := second.AwaitResult()
	if errA != nil || errB != nil {
		t.Fatalf("unexpected errors: %v, %v", errA, errB)
	}
	if calls.Load() != 2 {
		t.Fatalf("the function ran %d times for two runs", calls.Load())
	}
	got := []int64{a[0][0].(int64), b[0][0].(int64)}
	sort.Slice(got, func(i, j int) bool { return got[i] < got[j] })
	if !reflect.DeepEqual(got, []int64{1, 2}) {
		t.Errorf("the two runs returned %v, want one result each (1 and 2)", got)
	}
}

// Run with -race: runs of one group must not share where they store results.
func TestConcurrentRunsKeepTheirOwnResults(t *testing.T) {
	g := parallel.GroupUp(
		func() string { return "a" },
		func() string { return "b" },
		func() string { return "c" },
	)
	want := [][]any{{"a"}, {"b"}, {"c"}}

	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results, err := g.Run().AwaitResult()
			if err != nil || !reflect.DeepEqual(results, want) {
				t.Errorf("results %v, error %v; want %v", results, err, want)
			}
		}()
	}
	wg.Wait()
}

func TestGroupUpCopiesItsArguments(t *testing.T) {
	fns := []any{func() string { return "original" }}
	g := parallel.GroupUp(fns...)
	fns[0] = func() string { return "replaced" }

	results, err := g.Run().AwaitResult()
	if err != nil || results[0][0] != "original" {
		t.Errorf("results %v, error %v; the group changed when the caller's slice did", results, err)
	}
}

func TestAwaitingARunTwiceGivesTheSameOutcome(t *testing.T) {
	r := parallel.GroupUp(func() int { return 7 }, func() { panic("boom") }).Run()
	a, errA := r.AwaitResult()
	b, errB := r.AwaitResult()
	if !reflect.DeepEqual(a, b) {
		t.Errorf("first await %v, second await %v", a, b)
	}
	if errA == nil || errB == nil || errA.Error() != errB.Error() {
		t.Errorf("first error %v, second error %v", errA, errB)
	}
	if err := r.AwaitNoResult(); err == nil || err.Error() != errA.Error() {
		t.Errorf("AwaitNoResult returned %v, want %v", err, errA)
	}
}

func TestAnEmptyGroup(t *testing.T) {
	results, err := parallel.GroupUp().Run().AwaitResult()
	if err != nil || len(results) != 0 {
		t.Errorf("results %v, error %v; want no results and no error", results, err)
	}
}

func TestAFunctionsOwnErrorIsAResult(t *testing.T) {
	mine := errors.New("mine")
	results, err := parallel.GroupUp(func() (int, error) { return 0, mine }).Run().AwaitResult()
	if err != nil {
		t.Fatalf("a returned error was reported as a failure: %v", err)
	}
	if len(results[0]) != 2 || results[0][0] != 0 || results[0][1] != mine {
		t.Errorf("slot %v, want [0 mine]", results[0])
	}
}

func TestAPanicIsAWorkerError(t *testing.T) {
	results, err := parallel.GroupUp(
		func() int { return 1 },
		func() int { panic("boom") },
		func() int { return 3 },
	).Run().AwaitResult()

	if results[0][0] != 1 || results[2][0] != 3 {
		t.Errorf("the functions that finished lost their results: %v", results)
	}
	if results[1] != nil {
		t.Errorf("the panicking function's slot is %v, want nil", results[1])
	}
	failures := workerErrors(t, err)
	if len(failures) != 1 {
		t.Fatalf("got %d failures, want 1: %v", len(failures), err)
	}
	we := failures[0]
	if we.Index != 1 || we.Panic != "boom" || we.Err != nil {
		t.Errorf("WorkerError %+v, want index 1, panic \"boom\", no Err", we)
	}
	if len(we.Stack) == 0 {
		t.Error("the WorkerError carries no stack")
	}
	if !strings.Contains(err.Error(), "1") || !strings.Contains(err.Error(), "boom") {
		t.Errorf("the message %q does not say which function panicked with what", err)
	}
}

func TestAPanicWithAnErrorUnwrapsToIt(t *testing.T) {
	sentinel := errors.New("sentinel")
	_, err := parallel.GroupUp(func() { panic(sentinel) }).Run().AwaitResult()
	if !errors.Is(err, sentinel) {
		t.Errorf("errors.Is(%v, sentinel) is false", err)
	}
}

func TestValuesThatCannotBeCalled(t *testing.T) {
	var nilFunc func() int
	results, err := parallel.GroupUp(
		42,
		nil,
		nilFunc,
		func(x int) int { return x },
		func(xs ...int) int { return len(xs) },
	).Run().AwaitResult()

	for i, slot := range results {
		if slot != nil {
			t.Errorf("slot %d is %v, want nil", i, slot)
		}
	}
	failures := workerErrors(t, err)
	if len(failures) != 5 {
		t.Fatalf("got %d failures, want 5: %v", len(failures), err)
	}
	for i, we := range failures {
		if we.Index != i || we.Err == nil || we.Panic != nil {
			t.Errorf("failure %d: %+v, want index %d with Err and no panic", i, we, i)
		}
	}
}

func TestEveryFailureIsReportedInOrder(t *testing.T) {
	_, err := parallel.GroupUp(
		func() { panic("first") },
		func() {},
		func() { panic("second") },
	).Run().AwaitResult()

	failures := workerErrors(t, err)
	if len(failures) != 2 || failures[0].Index != 0 || failures[1].Index != 2 {
		t.Fatalf("failures %+v, want indexes 0 and 2", failures)
	}
}

// Awaiting a group that never ran must not compile, and a run must not be run
// again; both follow from which type has which method.
func TestOnlyARunCanBeAwaited(t *testing.T) {
	group := reflect.TypeOf(&parallel.ParallelGroup{})
	for _, name := range []string{"AwaitResult", "AwaitNoResult"} {
		if _, ok := group.MethodByName(name); ok {
			t.Errorf("*ParallelGroup has %s", name)
		}
	}
	if _, ok := group.MethodByName("Run"); !ok {
		t.Error("*ParallelGroup has no Run")
	}
	run := reflect.TypeOf(&parallel.RunningGroup{})
	if _, ok := run.MethodByName("Run"); ok {
		t.Error("*RunningGroup has Run")
	}
}
