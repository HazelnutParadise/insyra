# Proposal: core-slog-logger

## Why

api-review K-17 (#215), ported from `0.4` to the 0.3.x line because it only adds API: insyra's log lines can only go to the standard `log` package. Measured on 2026-10-13 on `0.4` at 4daa9af0, none of the three ways a program has today keeps what a log line is:

- `log.SetOutput(w)` redirects insyra, and every other line the application writes through `log` with it.
- `slog.SetDefault(...)` routes insyra's lines through the application's handler, but as plain `log` output, so every one arrives at `INFO`: `DataTable.GetCol("Z")`'s error became `{"level":"INFO","msg":"\u001b[31m[insyra - Error] DataTable.GetCol: column Z does not exist, the table has 1 column(s)\n\u001b[0m"}`, colour codes included.
- `Config.SetDefaultErrHandlingFunc` receives warnings, errors and fatals, never Info or Debug, on a goroutine of its own.

The owner ruled on 2026-10-13 to add a `*slog.Logger` hook and keep everything else as it is ([comment](https://github.com/HazelnutParadise/insyra/issues/215#issuecomment-6146697581)). `log/slog` is the standard library's common logging backend: zap, zerolog and others provide slog handlers, and `slog.New(slog.NewTextHandler(w, nil))` covers a plain `io.Writer`, so an `io.Writer` setter was rejected as the weaker of the two. Making Warning the default level was rejected in the same ruling; `quiet-success-logs` handles the success lines instead.

## What Changes

- `Config.SetLogger(l *slog.Logger)` sends every line insyra logs to `l` as a record:
  - **Level:** the line's own level. Debug, Info and Warning map to slog's Debug, Info and Warn, and Fatal to `slog.LevelError + 4`. This line has no Error level.
  - **Message:** the formatted text, without the `[insyra - …]` prefix, colour codes or trailing newline.
  - **Attributes:** `package` and `func`, when the call names them.
  - **Source:** the record's source is the function that logged, not insyra's logger.
- `SetLogger(nil)` restores the default output. `Config.GetLogger()` returns the logger set, or nil.
- `Config.SetLogLevel` still decides which lines insyra emits. The handler then decides which of them it keeps, so a handler's own minimum level filters on top.
- With no logger set, nothing changes: the same text, colours and destination (the standard `log` package).
- `SetDefaultConfig` clears the logger, like every other `Config` field.
- **A line outside the `Log…` functions:** CCL's warning when it cannot restore a row index took the standard `log` directly. It now takes the same path, through a hook that `internal/ccl` exposes and the root package sets at init; it is also recorded in the error buffer and handed to the error hook, like any warning.
- **`LogFatal`:** still ends the program after the record is handled, unless `SetDontPanic(true)` is on.
- **Unchanged:** the error buffer, `SetDefaultErrHandlingFunc`, `SetPanicOnError`, and the rule that a recorded error is logged at Warning level.

## Capabilities

### New Capabilities

- `core-log-output`: where insyra's log lines go, and what a program can attach to receive them.

### Modified Capabilities

None.

## Impact

- `logger.go`, `config.go`, `init.go`, `internal/ccl/ccl_evaluator.go` plus a new hook file in `internal/ccl`, and a new root test file.
- `Docs/Configuration.md` and both changelogs (`### Core`).
- **Compatibility:** the API only grows. `SetLogger` and `GetLogger` are methods on `Config`'s unexported type, so no caller has to implement anything, and the default output is byte-for-byte what it was.
