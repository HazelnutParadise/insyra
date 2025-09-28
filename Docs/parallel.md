# [ parallel ] Package

> **Note:** When using the `parallel` package, you may encounter editor warnings related to type assertions or function signatures. Rest assured, these warnings are expected due to the use of reflection and generic function signatures in Go. They do not affect the performance or correctness of your code and can be safely ignored.

The `parallel` package provides a simple and efficient way to execute multiple functions concurrently in Go. This package is designed to help you leverage parallel computing without needing specialized knowledge.

## Installation

To install the `parallel` package, use the following command:

```bash
go get github.com/HazelnutParadise/insyra/parallel
```

## Usage

The `parallel` package is easy to use and integrate into your projects. Below is an example demonstrating how to store functions in variables and run them in parallel.

### Example

```go
package main

import (
 "fmt"
 "github.com/HazelnutParadise/insyra/parallel"
)

func main() {
 // Define functions and store them in variables
 f1 := func() (int, string) { return 42, "Answer to Everything" }
 f2 := func() (string, int) { return "Hello, World!", 2024 }
 f3 := func() ([]int, float64) { return []int{1, 2, 3}, 3.14 }

 // Group the functions and run them in parallel
 pg := parallel.GroupUp(f1, f2, f3).Run()

 // Await results
 results := pg.AwaitResult()

 // Print the results
 fmt.Printf("All tasks completed. Results: %v\n", results)
 for i, result := range results {
  fmt.Printf("Task %d: %v\n", i, result)
 }
}
```

#### Example with No Return Values

For functions that do not return values, you can use `AwaitNoResult` for better performance:

```go
package main

import (
    "fmt"
    "github.com/HazelnutParadise/insyra/parallel"
)

func main() {
    counter := 0
    f1 := func() { counter++ }
    f2 := func() { counter += 2 }

    // Group the functions and run them in parallel
    pg := parallel.GroupUp(f1, f2).Run()

    // Await completion without collecting results
    pg.AwaitNoResult()

    fmt.Printf("Counter: %d\n", counter) // Output: Counter: 3
}
```

### How It Works

1. **GroupUp:** This function initializes a new `ParallelGroup` with any number of functions that return arbitrary types. These functions can be stored in variables for clarity and ease of reuse.

2. **Run:** This method starts the execution of all functions in parallel. For performance optimization, it only collects results for functions that actually return values.

3. **AwaitResult:** This method waits for all functions to complete and then returns their results in a `[][]interface{}` format. For functions with no return values, the corresponding result will be `nil`.

4. **AwaitNoResult:** This method waits for all functions to complete without returning any results. It is optimized for scenarios where you only need to ensure completion, avoiding the overhead of result collection.

### Key Features

- **Helps You with Parallel Computation:** The `parallel` package simplifies parallel computation in Go, making it accessible even if you don't have specialized knowledge in concurrency.
  
- **No Need for Advanced Knowledge:** You don't need to be an expert in Go's concurrency model to use this package effectively.

- **Optimized for Different Use Cases:** Use `AwaitResult` when you need the return values, or `AwaitNoResult` for better performance when results are not needed.

### Handling Editor Warnings

**Important:** When using the `parallel` package, your editor might display warnings related to type assertions or function signatures. This is expected due to the use of reflection and generic function signatures in Go. You can safely ignore these warnings, as they do not impact the performance or correctness of your code.

### Benefits

- **Simplicity:** This package is designed to be straightforward and easy to use.
- **Efficiency:** It enables parallel execution of tasks, which can improve performance, especially for I/O-bound or independent tasks.
- **Flexibility:** It supports functions with varying return types and signatures.

### Limitations

- The package expects functions to return arbitrary values, which are stored as `[]interface{}`. You may need to manage type assertions accordingly.
- Type safety is managed through runtime checks, which may cause editor warnings.
- When using `AwaitResult` with functions that have no return values, the corresponding results will be `nil`.
