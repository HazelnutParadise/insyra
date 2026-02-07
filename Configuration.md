# Configuration

Insyra provides a global `Config` object for managing library behavior. You can customize logging, error handling, and performance settings.

**Default values (after `SetDefaultConfig`)**:

- Log level: `LogLevelInfo`
- Colored output: `true`
- Panic protection: `false`
- Thread safety: `true`

## Log Level Management

Control what level of messages are logged:

```go
// Set log level - only messages at this level or above will be logged
insyra.Config.SetLogLevel(insyra.LogLevelDebug)    // Most verbose
insyra.Config.SetLogLevel(insyra.LogLevelInfo)     // Default
insyra.Config.SetLogLevel(insyra.LogLevelWarning)  // Only warnings and errors
insyra.Config.SetLogLevel(insyra.LogLevelFatal)    // Only fatal errors

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

## Error Handling (global)

Configure how errors are handled globally across Insyra:

```go
// Prevent panics and handle errors gracefully instead
insyra.Config.SetDontPanic(true)

// Check panic prevention status
isPanicPrevented := insyra.Config.GetDontPanicStatus()

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

**Note:** For chainable methods on `DataList` and `DataTable`, you can inspect and clear instance-level errors using `Err()` and `ClearErr()` (e.g., `dl.Err()`, `dt.Err()`).

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
    insyra.Config.SetDontPanic(true)
    
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
