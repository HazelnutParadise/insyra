# Proposal: fix-api-review-batch-3

## Why

After the two bug batches, the review still lists a set of items that need no API decision but do affect production use: a chart export that uploads the user's data to a third-party service by default, a global error hook that spawns one goroutine per warning, `Config` setters that race their hot-path readers, an encoding sniffer that mis-detects UTF-8 when a multi-byte character straddles its 8 KB sample, two O(n²) cleanup methods, an equality test that says a list is never equal to its own clone when it holds NaN, unguarded type assertions on interface parameters, a command registry mutated without a lock, and I/O helpers that wrap errors with `%v`, create world-writable directories, write output files non-atomically, or log through the standard logger instead of Insyra's. Each has one obviously right behaviour.

## What Changes

- `plot.SavePNG` no longer falls back to the online rendering service unless the caller passes `true`; the doc says what the fallback sends where. **BREAKING (behavioural)**: callers relying on the silent fallback must opt in.
- The default error-handling hook set by `Config.SetDefaultErrHandlingFunc` runs on one dedicated goroutine fed by a bounded channel, in order, instead of one goroutine per error. `Config`'s log level, colour, and dont-panic fields are atomic, so a setter never races `LogInfo`. The "Welcome to Insyra" banner is no longer printed on import; the CLI REPL prints it itself.
- `insyra.DetectEncoding` backs the UTF-8 check off to the last complete rune before validating, so a character cut by the sample boundary is not mistaken for another encoding.
- `DataList.ClearNaNs`/`ClearNils`/`ClearNilsAndNaNs` filter in one pass; `DropAll` and `ClearStrings` drop their per-call goroutine fan-out. `IsEqualTo` treats NaN as equal to NaN (pandas `equals` semantics). `Update` records its own name in `Err()`.
- `DataTable.FindColsIfContains`/`FindColsIfContainsAll` no longer push a warning into `Err()` for every non-matching column; `containsSubstring` is `strings.Contains`; `Count` and `Clone` drop their goroutine fan-out. `AppendRowsByColName` adds new columns in sorted key order, so `ReadJSON` produces the same column order on every run instead of following Go's randomized map iteration.
- `csvxl` wraps errors with `%w` and creates directories with 0755. `parquet.Write` writes to a temp file and renames; all `parquet` close-error logging goes through `insyra.LogWarning`.
- Every `stats` function that took `insyra.IDataList` and asserted `*insyra.DataList` converts through one helper that accepts any implementation and reports nil as an error instead of panicking.
- `cli/commands.Registry` access is guarded by a mutex; `mkt` default-value notices drop from Info to Debug.
- Docs: garbled comments in `atomic.go`, stale op names in `accel`, the `ExecuteRequest`/`WorkloadEstimate` docs, `SetDefaultConfig`'s doc, `MovingStdev`/`Drop`/`ToF64Slice`/`Len` docs, `ApplyCCL`'s example, `parallel.AwaitNoResult`'s claim, `ReadSQLOptions.WhereClause`'s injection warning, `ToSQLOptions.IfExists`'s stale comment.
- Both changelogs and `api-review.md` (rows marked fixed) are updated in the same change.

## Capabilities

### New Capabilities

- `plot-png-export-policy`: `SavePNG` never sends data off-host without an explicit opt-in.
- `core-config-and-error-hook-safety`: config fields are atomic; the error hook is bounded and ordered; importing the library prints nothing.
- `core-encoding-detection`: a UTF-8 file is detected as UTF-8 regardless of where the sample boundary falls.
- `datalist-equality-and-cleanup`: NaN-aware equality; single-pass cleanup; correct function name in `Err()`.
- `datatable-find-hygiene`: Find* helpers do not record warnings for non-matches; row maps add columns in a deterministic order.
- `io-error-hygiene`: `%w` wrapping, 0755 directories, atomic `parquet.Write`, Insyra logger.
- `stats-input-type-guard`: interface parameters are converted, never asserted; nil is an error.
- `cli-registry-safety`: the command registry is safe for concurrent registration and dispatch.

### Modified Capabilities

(none)

## Impact

- `plot/save_chart.go`, `config.go`, `error_buffer.go`, `init.go`, `cli/repl/repl.go`, `utils.go`, `datalist.go`, `datatable.go`, `csvxl/convert.go`, `csvxl/convertDir.go`, `parquet/*.go`, `stats/*.go` (13 assertion sites), `cli/commands/registry.go`, `mkt/*.go`, `atomic.go`, `accel/types.go`, `accel/executor.go`, `datatable_to_sql.go`, `datatable_from_sql.go`, `parallel/parallel_computing.go`, plus tests, docs and changelogs.
- No signature changes; no new dependencies.
