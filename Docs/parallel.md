# [ parallel ] Package

The `parallel` package runs several functions at the same time and collects what each one returns. You put each piece of work in a function that takes no arguments, and the package starts the goroutines, waits for them, and tells you if any of them failed. You do not need to know Go's concurrency primitives to use it.

## Installation

```bash
go get github.com/HazelnutParadise/insyra/parallel
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"
    "slices"

    "github.com/HazelnutParadise/insyra/parallel"
)

func main() {
    sales := []float64{120, 260, 40, 420, 310}

    total := func() float64 {
        sum := 0.0
        for _, v := range sales {
            sum += v
        }
        return sum
    }
    lowHigh := func() (float64, float64) { return slices.Min(sales), slices.Max(sales) }
    count := func() int { return len(sales) }

    results, err := parallel.GroupUp(total, lowHigh, count).Run().AwaitResult()
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(results[0][0])                // 1150
    fmt.Println(results[1][0], results[1][1]) // 40 420
    fmt.Println(results[2][0])                // 5
}
```

## Functions and Types

### GroupUp

```go
func GroupUp(fns ...any) *ParallelGroup
```

**Description:** Collects the functions to run. Each one must take no arguments and may return any number of values of any types. To pass inputs, capture them in a closure: `func() float64 { return score(data) }`. The list is copied, so changing the slice you passed afterwards does not change the group.

### ParallelGroup.Run

```go
func (pg *ParallelGroup) Run() *RunningGroup
```

**Description:** Starts every function in its own goroutine and returns this run. A group can be run as many times as you like, and each `Run` is a separate run with its own results. Because a closure reads its captured variables when it runs, running a group again after the data changed computes with the new data.

When two runs of the same group overlap, each function executes twice at once, so it must not write to a variable another call of it also writes.

### RunningGroup.AwaitResult

```go
func (r *RunningGroup) AwaitResult() ([][]any, error)
```

**Description:** Waits for every function in the run and returns what each one returned, in the order they were passed to `GroupUp`.

**Returns:**

- `[][]any`: One slot per function. A slot holds the function's return values, including an `error` the function returned itself. It is `nil` for a function with no return values and for a function that failed.
- `error`: `nil` when every function finished. Otherwise it holds a `*WorkerError` for each function that panicked or could not be called. See [When a Function Fails](#when-a-function-fails).

`AwaitResult` and `AwaitNoResult` exist only on a run, so awaiting a group you never started does not compile. A run can be awaited more than once and gives the same results each time.

Because `AwaitResult` returns two values, Go does not allow `a, b := r1.AwaitResult(), r2.AwaitResult()`. Await each run on its own line.

### RunningGroup.AwaitNoResult

```go
func (r *RunningGroup) AwaitNoResult() error
```

**Description:** Waits for every function in the run and returns the same error as `AwaitResult`, without the return values. Use it when the functions store their results themselves:

```go
sales := []float64{120, 260, 40, 420, 310}
var low, high float64

err := parallel.GroupUp(
    func() { low = slices.Min(sales) },
    func() { high = slices.Max(sales) },
).Run().AwaitNoResult()
if err != nil {
    log.Fatal(err)
}
fmt.Println(low, high) // 40 420
```

Each function writes its own variable, and the variables are read only after `AwaitNoResult` returns. Two functions that write the same variable would race; use `sync/atomic` or a mutex for a value they share.

## When a Function Fails

A function that panics does not crash your program. Its slot is `nil`, the other functions keep their results, and the error from `AwaitResult` or `AwaitNoResult` describes what happened as a `*WorkerError`:

```go
type WorkerError struct {
    Index int    // the function's position among GroupUp's arguments
    Panic any    // the value it panicked with, or nil
    Stack []byte // the goroutine's stack at the panic, or nil
    Err   error  // why the value could not be called, or nil
}
```

The same error is returned for a value that cannot be called: something that is not a function, a nil function, or a function that takes arguments. Its `Err` says which.

```go
results, err := parallel.GroupUp(
    func() int { return 1 },
    func() int { panic("out of range") },
).Run().AwaitResult()

var we *parallel.WorkerError
if errors.As(err, &we) {
    fmt.Println("function", we.Index, "panicked:", we.Panic) // function 1 panicked: out of range
}
fmt.Println(results[0], results[1] == nil) // [1] true
```

When several functions fail, the error joins one `*WorkerError` per function in order, and its message has one line for each. When a function panics with an `error` value, `errors.Is(err, thatError)` finds it.

An `error` that a function returns normally is not a failure. It is one of the function's return values and stays in its slot.
