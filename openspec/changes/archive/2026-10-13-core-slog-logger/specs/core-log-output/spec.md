## ADDED Requirements

### Requirement: A program can send insyra's log lines to a slog logger

`Config.SetLogger(l)` with a non-nil `*slog.Logger` SHALL send every line insyra logs to `l` as one record. While a logger is set, insyra SHALL write nothing to the standard `log` package.

#### Scenario: Nothing reaches the standard log
- **WHEN** a logger is set and `LogWarning` is called
- **THEN** the handler receives the record
- **AND** nothing is written to the standard `log` package

### Requirement: A record keeps the line's level

Debug, Info, Warning and Error SHALL map to `slog.LevelDebug`, `slog.LevelInfo`, `slog.LevelWarn` and `slog.LevelError`. Fatal SHALL map to `slog.LevelError + 4`.

#### Scenario: Each level keeps its level
- **WHEN** the log level is Debug, a logger is set, and `LogDebug`, `LogInfo`, `LogWarning`, `LogError` and `LogFatal` are each called
- **THEN** the handler receives five records with levels Debug, Info, Warn, Error and Error+4

#### Scenario: A recorded error arrives at Error level
- **WHEN** a logger is set and `GetCol("Z")` is called on a one-column table
- **THEN** the handler receives one Error record from `DataTable.GetCol`

### Requirement: A record carries the message and where it came from

A record's message SHALL be the formatted text, without the `[insyra - …]` prefix, ANSI colour codes or a trailing newline. It SHALL carry a `package` and a `func` attribute, each only when the call gives it a non-empty value. Its program counter SHALL be that of the function that called the logger.

#### Scenario: Message and attributes
- **WHEN** a logger is set and `LogInfo("pkg", "Fn", "value %d", 7)` is called
- **THEN** the record's message is `value 7`, with attributes `package=pkg` and `func=Fn`

#### Scenario: No text format in the message
- **WHEN** a logger is set and `GetCol("Z")` is called on a one-column table
- **THEN** the record's message contains no ANSI escape and no `[insyra` prefix

#### Scenario: Empty names add no attributes
- **WHEN** `LogInfo("", "", "hello")` is called with a logger set
- **THEN** the record has message `hello` and no attributes

#### Scenario: The source is the caller
- **WHEN** a test function calls `LogInfo` with a logger set
- **THEN** the record's program counter resolves to that test function

### Requirement: insyra's level still decides what is emitted

With a logger set, `Config.SetLogLevel` SHALL filter lines exactly as it does without one. The handler SHALL see only the lines that pass, and its own minimum level SHALL filter on top.

#### Scenario: Info is not emitted at Warning level
- **WHEN** the log level is Warning, a logger whose handler accepts every level is set, and `LogInfo` and then `LogWarning` are called
- **THEN** the handler receives only the warning

### Requirement: The default output is unchanged

With no logger set, every line SHALL go to the standard `log` package in the format and colours it had before `SetLogger` existed. `SetLogger(nil)` SHALL restore that output. `Config.GetLogger()` SHALL return the logger set, or nil. `SetDefaultConfig()` SHALL clear the logger.

#### Scenario: Clearing the logger
- **WHEN** a logger is set, then `SetLogger(nil)` is called, then `LogWarning("pkg", "Fn", "careful")`
- **THEN** `GetLogger()` returns nil
- **AND** the standard `log` package receives `[insyra - Warning] pkg.Fn: Careful`

#### Scenario: Resetting the configuration
- **WHEN** a logger is set and `SetDefaultConfig()` is called
- **THEN** `GetLogger()` returns nil

### Requirement: Lines written outside the logger functions take the same path

The error hook's queue-full warning and CCL's row-restore warning SHALL be emitted at Warning level through the same path as `LogWarning`, so a set logger receives them. The queue-full warning SHALL NOT be recorded in the error buffer or handed to the error hook.

#### Scenario: The hook queue fills up
- **WHEN** a logger is set, the error hook blocks, and more warnings are recorded than the hook queue holds
- **THEN** the handler receives one Warn record saying the queue is full

#### Scenario: CCL warns through insyra
- **WHEN** a logger is set and `internal/ccl`'s warning hook is called with function `evaluate` and a message
- **THEN** the handler receives one Warn record with `package=ccl` and that message

### Requirement: Setting the logger is safe while logging

`SetLogger` and `GetLogger` SHALL be safe to call while other goroutines log. `go test -race` SHALL NOT report a data race.

#### Scenario: Concurrent set and log
- **WHEN** one goroutine repeatedly sets and clears the logger while another repeatedly calls `LogInfo`
- **THEN** the race detector reports nothing
