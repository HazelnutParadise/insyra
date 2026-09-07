# Configuration

Insyra provides a global `Config` object for managing library behavior. You can customize logging, error handling, and performance settings.

**Default values (after `SetDefaultConfig`)**:

- Log level: `LogLevelInfo`
- Colored output: `true`
- Panic protection: `false`
- Thread safety: `true`
- Acceleration: `true`

## Log Level Management

Control what level of messages are logged:

```go
// Set log level - only messages at this level or above will be logged
insyra.Config.SetLogLevel(insyra.LogLevelDebug)    // Most verbose
insyra.Config.SetLogLevel(insyra.LogLevelInfo)     // Default
insyra.Config.SetLogLevel(insyra.LogLevelWarning)  // Warnings and above
insyra.Config.SetLogLevel(insyra.LogLevelError)    // Only failures
insyra.Config.SetLogLevel(insyra.LogLevelFatal)    // Only fatal failures

// Get current log level
level := insyra.Config.GetLogLevel()
```

## Colored Output

Control whether terminal output is colored:

```go
// Enable / disable colored output
insyra.Config.SetUseColoredOutput(true)

// Check colored output status
usesColor := insyra.Config.GetDoesUseColoredOutput()
```

## Error Handling

Insyra never ends or interrupts your program. A failure is recorded and the
call returns something usable, so you decide when and how to react. Errors
reach you in exactly two shapes:

| Shape | Where |
| --- | --- |
| A returned `error` | ordinary functions: `stats`, `quant`, `csvxl`, `parquet`, `gplot.SaveChart`, file readers |
| A sticky `Err()` on the value | the chainable types: `DataList`, `DataTable`, and the `isr` wrappers |

`Err()` is **sticky**: the first failure stays until you clear it, so a long
chain reports the root cause rather than the last symptom. Read and clear in
one step with `PopErr()`:

```go
result := dl.Sort().Normalize().MovingAverage(3)
if err := result.PopErr(); err != nil {
    // handle it; `result` is a usable (empty) list either way
}
```

A lookup that simply finds nothing — a missing column name, a value that is
not in the list, a statistic over an empty list — is a normal result, not an
error, and does not touch `Err()`. An out-of-range index does.

### Failing fast (opt-in)

For a script or a notebook, stopping at the first mistake can beat carrying on
with empty data. `SetPanicOnError(true)` turns every recorded error into a
`panic` carrying an `*insyra.ErrorInfo` (which implements `error`). It is a
panic, never `os.Exit`, so you can still recover.

```go
// Default: nothing panics, nothing exits. Check Err() / the returned error.
insyra.Config.SetPanicOnError(false)

// Opt in to fail-fast.
insyra.Config.SetPanicOnError(true)

// Read the current setting.
failFast := insyra.Config.GetPanicOnError()
```

> `SetDontPanic` / `GetDontPanicStatus` are the old inverse of this switch.
> They still work and will be removed in the release after next; use
> `SetPanicOnError`.

### Watching everything that happened (global)


// Set custom error handling function for all errors
insyra.Config.SetDefaultErrHandlingFunc(func(errType insyra.LogLevel, packageName, funcName, errMsg string) {
    // Your custom error handling logic
    // errType: The severity level of the error
    // packageName: The package where the error occurred
    // funcName: The function where the error occurred
    // errMsg: The error message
    // Use %v to print LogLevel values reliably
    fmt.Printf("[%v] %s.%s: %s\n", errType, packageName, funcName, errMsg)
})

// Get the current error handling function
handler := insyra.Config.GetDefaultErrHandlingFunc()
```

The global buffer behind `GetAllErrors`, `PopAllErrors`, `HasError`,
`GetErrorCount` and `ClearErrors` is a **diagnostic log**, not an
error-handling API: it holds up to 1536 records from every goroutine and every
object mixed together, dropping the oldest when full. Use it to see what a run
did; handle errors through `Err()`/`PopErr()` or the returned `error`.

## Performance Configuration

Fine-tune performance for your use case:

```go
// DANGER: Turn off thread safety for extreme performance
// Use ONLY when you are sure there are no concurrent accesses.
// Data consistency is NOT guaranteed when this is disabled.
insyra.Config.Dangerously_TurnOffThreadSafety()

// If you need to reset all configs back to library defaults, call:
insyra.SetDefaultConfig()
```

Control device acceleration independently from thread safety:

```go
insyra.Config.SetAcceleration(false) // use the exact CPU paths
enabled := insyra.Config.GetAccelerationEnabled()
insyra.Config.SetAcceleration(true)
```

Acceleration is enabled by default. `INSYRA_ACCEL_DISABLE_WGPU=1` is the
operations override for the builtin WebGPU backend and wins over Config, so
the backend remains disabled while that environment variable is set.

> Thread safety (on by default) serializes access to **one** instance per `AtomicDo`.
> To operate on **several** instances atomically, use `insyra.AtomicDoAll(func(){ ... }, a, b, ...)`,
> which locks all the given `DataList`/`DataTable` instances together in a deadlock-free order.
> Do NOT nest `AtomicDo` on a different instance to read two of them — the inner call does not
> lock the other instance and can race a concurrent mutation.

## Complete Example

```go
package main

import (
    "fmt"
    "github.com/HazelnutParadise/insyra"
)

func main() {
    // Initialize with custom configuration
    insyra.Config.SetLogLevel(insyra.LogLevelDebug)
    insyra.Config.SetPanicOnError(true) // fail fast in this script
    
    // Custom error handler
    insyra.Config.SetDefaultErrHandlingFunc(func(errType insyra.LogLevel, pkg, fn, msg string) {
        fmt.Printf("ERROR in %s.%s: %s\n", pkg, fn, msg)
    })
    
    // Now use Insyra with these settings
    dl := insyra.NewDataList(1, 2, 3, 4, 5)
    fmt.Println(dl.Mean())
}
```

For implementation details, see the [config.go](../config.go) source file.
