# Proposal: make-errors-non-terminating

## Why

Insyra has four error channels at once — a returned `nil`, an empty object, the instance `Err()`, and a global buffer fed by `LogFatal` — and `LogFatal` ends the host process by `os.Exit(1)` at 33 call sites. A library must never terminate its caller, and a caller must not have to guess which of four channels a given function uses. The opt-out `SetDontPanic(true)` does not fix it either: `LogFatal` then returns and the next line dereferences a nil, so "don't panic" is not true today.

The decided philosophy: **the library never terminates or panics by default; panicking is opt-in.** Errors reach the caller through exactly two shapes — a returned `error` for ordinary functions, and a sticky `Err()` for the chainable types. This closes #205 (LogFatal), #206 (no error level), and #216 (three failure shapes on DataList).

## What Changes

- **`LogFatal` no longer exits.** It records the error and returns. `Config.SetPanicOnError(true)` is the opt-in that turns any recorded error — `LogFatal` and anything written to `Err()` — into a `panic` (never `os.Exit`, so the caller can recover). `SetDontPanic`/`GetDontPanicStatus` become Deprecated aliases for one release (follow-up already recorded in `AGENTS.md`).
- **New `LogLevelError`** between Warning and Fatal, and `LogError`. `Err()` now records at Error level; Warning goes back to meaning "done, but worth noticing".
- **`Err()` is sticky**: the first error stays until cleared, so a chain is checked once at the end and reports the root cause rather than the last symptom. New `PopErr()` reads and clears in one step. `ClearErr()` is unchanged. `Clone()` starts clean.
- **A lookup that finds nothing is not an error.** `Get` out of range, `FindFirst`/`FindLast` not found, `Count`/`FindAll` on an empty list, statistics over an empty list, and the other read-side "not found" paths log at Warning and leave `Err()` alone, so a sticky `Err()` cannot be polluted by normal results.
- **Chainable methods never return `nil`.** The nine `DataList` methods that returned `nil` on failure (`Normalize`, `MovingAverage`, `WeightedMovingAverage`, `ExponentialSmoothing`, `DoubleExponentialSmoothing`, `MovingStdev`, `Difference`, `Rank`) return an empty `DataList` carrying the error, and the error is also recorded on the receiver.
- **`isr` keeps its block syntax**: `DT.From`, `Col`, `Row`, `Push`, `UseDL`, `UseDT` return a usable object with `Err()` set instead of killing the process.
- **Ordinary functions return errors**: `gplot.SaveChart` returns `error` (**BREAKING**), the three `panic(err)` in `gplot` and the two in `plot` become recorded errors, `lp`'s install and temp-file failures propagate, `py`'s IPC listen failure is recorded.
- **The global error buffer is documentation-level demoted to diagnostics**: `GetAllErrors`, `PopAllErrors`, `ClearErrors`, `HasError`, `GetErrorCount` stay; the other nine are Deprecated. The ring is bounded at 1536 and drops the oldest instead of growing without limit (#207 / IN-7).

## Capabilities

### New Capabilities

- `error-philosophy`: the library never terminates or panics unless the caller opts in.
- `instance-error-contract`: `Err()` is sticky, `PopErr()` reads and clears, and only real failures are recorded.
- `chainable-never-nil`: a chainable method always returns a usable receiver or result.
- `global-error-buffer-scope`: the global buffer is a bounded diagnostic log, not an error-handling API.

### Modified Capabilities

(none)

## Impact

- `config.go`, `logger.go`, `error_buffer.go`, `datalist.go`, `datatable.go` and their `*_*.go` siblings, `interfaces.go`, `isr/*`, `gplot/*`, `plot/radar.go`, `plot/heatmap.go`, `lp/*`, `py/pyresult.go`, `cli/commands/timeseries.go`, docs, skills, both changelogs.
- **BREAKING**: `gplot.SaveChart` gains an `error` return; `LogFatal` no longer terminates; methods that returned `nil` now return an empty list.
