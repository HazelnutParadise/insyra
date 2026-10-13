# Configuration

Insyra provides a global `Config` object for managing library behavior. You can customize logging, error handling, and performance settings.

**Default values (after `SetDefaultConfig`)**:

- Log level: `LogLevelInfo`
- Logger: none, so log lines go to the standard `log` package
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

## Sending Logs to slog

By default insyra writes its log lines to the standard `log` package, as text such as `[insyra - Warning] DataList.FindFirst: Value not found, returning nil.`. To send them to your own logging instead, give insyra a `*slog.Logger`:

```go
import "log/slog"

insyra.Config.SetLogger(slog.New(slog.NewJSONHandler(os.Stderr, nil)))

insyra.NewDataTable(insyra.NewDataList(1)).GetCol("Z")
// {"time":"…","level":"ERROR","msg":"column Z does not exist, the table has 1 column(s)","package":"DataTable","func":"GetCol"}

insyra.Config.SetLogger(nil) // back to the standard log package
current := insyra.Config.GetLogger() // nil when no logger is set
```

Each line becomes one record:

| insyra level | slog level |
| --- | --- |
| `LogLevelDebug` | `slog.LevelDebug` |
| `LogLevelInfo` | `slog.LevelInfo` |
| `LogLevelWarning` | `slog.LevelWarn` |
| `LogLevelError` | `slog.LevelError` |
| `LogLevelFatal` | `slog.LevelError + 4` (shown as `ERROR+4`) |

- **Message:** the formatted message, without the `[insyra - …]` prefix, colour codes or a trailing newline.
- **Attributes:** `package` and `func` name where the line came from, and are left out when the line names neither.
- **Source:** the record's source is the insyra function that logged, so a handler created with `AddSource: true` shows that function rather than insyra's logger.
- **Filtering:** `SetLogLevel` still decides which lines insyra emits. Your handler then keeps the ones its own level allows. At insyra's default `LogLevelInfo`, a handler set to `slog.LevelDebug` still receives no Debug lines until you also call `SetLogLevel(insyra.LogLevelDebug)`.
- **Other packages:** zap, zerolog and most other logging packages provide a slog handler, so the same call connects insyra to them.
- **A plain `io.Writer`:** `slog.New(slog.NewTextHandler(w, nil))`.
- **Errors:** a recorded error is logged at Error level whether or not a logger is set. The error buffer and `SetDefaultErrHandlingFunc` below receive the same lines as before.

## Colored Output

Control whether terminal output is colored:

```go
// Enable / disable colored output
insyra.Config.SetUseColoredOutput(true)

// Check colored output status
usesColor := insyra.Config.GetUseColoredOutput()
```

`GetDoesUseColoredOutput` is the deprecated name of `GetUseColoredOutput`. It returns the same value and is removed in the release after the one that deprecated it.

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

Asking whether a value is present and getting "no" is an answer, not a
failure, so `FindFirst`, `FindLast`, `Count` and statistics over an empty list
leave `Err()` alone. Addressing a column, row or index that is not there is
recorded, because the caller asserted it existed and the only other signal
would be a bare `nil`.

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

```go
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

// Turn the locking back on, leaving every other setting as it is
insyra.Config.TurnOnThreadSafety()

// If you need to reset all configs back to library defaults, call:
insyra.SetDefaultConfig()
```

What turning it off removes is the lock each `DataList` and `DataTable` takes in `AtomicDo` and `AtomicDoAll`. Every method of those types, and of any type that embeds one (`isr`'s `DL` and `DT` included), is then unsafe to call from more than one goroutine at a time. `Config`, the global error buffer and logging stay safe, because they do not use those locks. Change the switch in either direction only while no other goroutine is using a `DataList` or a `DataTable`. `TurnOnThreadSafety` turns the locking back on without touching the log level, coloured output, the error hook or acceleration, which `SetDefaultConfig` would reset too.

The underscore in `Dangerously_TurnOffThreadSafety` is deliberate. Go names do not use one, so the call looks wrong wherever it is written, and that is the warning. Naming linters report only the declaration inside insyra, never your call.

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
