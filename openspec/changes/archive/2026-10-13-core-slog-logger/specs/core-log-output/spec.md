## ADDED Requirements

### Requirement: A program can send insyra's log lines to a slog logger

`Config.SetLogger(l)` with a non-nil `*slog.Logger` SHALL send every line insyra logs to `l` as one record. While a logger is set, insyra SHALL write nothing to the standard `log` package.

#### Scenario: Nothing reaches the standard log
- **WHEN** a logger is set and `LogWarning` is called
- **THEN** the handler receives the record
- **AND** nothing is written to the standard `log` package

### Requirement: A record keeps the line's level

Debug, Info and Warning SHALL map to `slog.LevelDebug`, `slog.LevelInfo` and `slog.LevelWarn`. Fatal SHALL map to `slog.LevelError + 4`.

#### Scenario: Each level keeps its level
- **WHEN** the log level is Debug, a logger is set, `SetDontPanic(true)` is on, and `LogDebug`, `LogInfo`, `LogWarning` and `LogFatal` are each called
- **THEN** the handler receives four records with levels Debug, Info, Warn and Error+4

#### Scenario: A recorded error arrives at Warn level
- **WHEN** a logger is set and `GetCol("Z")` is called on a one-column table
- **THEN** the handler receives one Warn record from `DataTable.GetCol`

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

### Requirement: CCL's warnings take the same path

CCL's row-restore warning SHALL be emitted at Warning level through the same path as `LogWarning`, so a set logger receives it, and SHALL be recorded in the error buffer and handed to the error hook like any other warning.

#### Scenario: CCL warns through insyra
- **WHEN** a logger is set and `internal/ccl`'s warning hook is called with function `evaluate` and a message
- **THEN** the handler receives one Warn record with `package=ccl` and that message

### Requirement: LogFatal still ends the program

With a logger set, `LogFatal` SHALL hand its record to the logger and then end the program, unless `Config.SetDontPanic(true)` is on, exactly as it does without a logger.

#### Scenario: Fatal with the safeguard on
- **WHEN** a logger is set, `SetDontPanic(true)` is on, and `LogFatal` is called
- **THEN** the handler receives one `ERROR+4` record and the program keeps running

### Requirement: Setting the logger is safe while logging

`SetLogger` and `GetLogger` SHALL be safe to call while other goroutines log. `go test -race` SHALL NOT report a data race.

#### Scenario: Concurrent set and log
- **WHEN** one goroutine repeatedly sets and clears the logger while another repeatedly calls `LogInfo`
- **THEN** the race detector reports nothing
