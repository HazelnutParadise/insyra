# Design: make-errors-non-terminating

## Context

Four error channels exist today (returned `nil`, empty object, instance `Err()`, global buffer + `os.Exit`). The goal is two: an `error` return for ordinary functions, and a sticky `Err()` for the chainable types. `isr` is deliberately a block-syntax API, so returning `error` from its methods is not an option — it must keep returning a chainable object.

## Decisions

1. **One switch, `Config.SetPanicOnError(bool)`.** Earlier drafting used `SetPanicOnFatal`; that name only covers `LogFatal`, but after this change `LogFatal` is just the top level of the same recording path, so a single switch covering every recorded error is simpler and cannot drift out of sync. It panics with the `*ErrorInfo` (which implements `error`), never `os.Exit`, so a caller can recover. `SetDontPanic(v)` becomes `SetPanicOnError(!v)` and is Deprecated; the `AGENTS.md` follow-up is updated to the final name.
2. **`LogLevelError` is inserted between Warning and Fatal.** `LogLevel` is an unexported-typed `int` constant block, so inserting a value shifts `LogLevelFatal` from 3 to 4. Nothing persists a `LogLevel` numerically (the CLI stores the string `"fatal"`), so this is safe; `String()` and the CLI parser gain the new level.
3. **Sticky `Err()`**: `setError` writes only when `lastError == nil`. The first failure survives every later step, so `dt.From(bad).Push(x).Sort(y).Err()` reports the read failure, not a downstream symptom. *Alternative*: keep last-error — rejected, it hides the root cause exactly when a chain is long.
4. **`PopErr()`** returns the error and clears it, so a chain ends with one statement and the object stays reusable. Added to `IDataList`/`IDataTable` (both already carry unexported methods, so no external implementer breaks).
5. **`warn` vs `fail`.** `warn` logs at Warning and does **not** touch `Err()`; `fail` logs at Error, records `Err()`, and honours `SetPanicOnError`. Each existing `warn` call site is classified by one rule: *did the caller ask for something impossible (bad argument, unreadable cell, a mutation that cannot be applied)?* → `fail`. *Did the call simply find nothing, or run over an empty list?* → `warn`. Without this split a sticky `Err()` would be permanently set by ordinary lookups.
6. **Never return `nil` from a chainable method.** The eight failing `DataList` transforms return `NewDataList()` (empty) with the error recorded on both the result and the receiver. *Alternative*: return a same-length list of `nil` — rejected here because it hides the failure behind plausible-looking data; D-18 tracks that question for the window reducers separately.
7. **`isr`**: `From`/`Col`/`Row`/`Push`/`UseDL`/`UseDT` build an empty `dt`/`dl` and record the error on it. `UseDL`/`UseDT` cannot return `nil` any more, so no caller can nil-deref a converted value.
8. **`gplot.SaveChart` returns `error`** — the only breaking signature change; a chart writer that cannot report a failed write is unusable in a pipeline. The three `plotter.New*` `panic(err)` calls and `plot`'s two calendar-mode panics become recorded errors returning `nil` charts, matching the rest of `plot`.
9. **`lp`**: the install path already runs under `sync.Once` inside `initGLPK`; failures now record an error and leave `glpsol` unfound, so `Solve*` reports "GLPK not available" through its existing additional-info table instead of the process dying at import-adjacent time.
10. **Global buffer**: bounded push (drop oldest at 1536) and a documentation demotion. Nine of the fourteen accessors are Deprecated rather than deleted so nothing breaks in this release; the deletion is a recorded follow-up.

## Risks / Trade-offs

- [A script that relied on `LogFatal` stopping the program now continues past the failure] → BREAKING, called out in both changelogs with the one-line fix (`SetPanicOnError(true)`).
- [Sticky `Err()` surprises someone reusing an object] → `Clone()` starts clean, `PopErr()`/`ClearErr()` reset, and every doc page states the rule.
- [`LogLevelFatal` changes numeric value] → no persisted numeric use; the CLI and config work in strings.
- [`gplot.SaveChart` signature] → callers must handle or ignore the error; docs and skills updated in the same change.
