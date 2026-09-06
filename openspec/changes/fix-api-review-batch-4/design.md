# Design: fix-api-review-batch-4

## Context

Every item was reproduced by the second-round review and re-verified before this change (CCL-1, CCL-2, IN-1, IN-3, CLI-1 root cause, TS-1, RP-1, SEC-2, SEC-4). The rule for inclusion: one obviously right behaviour, no exported signature change, no reversal of a documented contract.

## Decisions

1. **CCL out-of-range references** — checked once per expression after `Bind` with `MaxResolvedColIndex`, at all three entry points, rather than changing `Context.GetCol` (public through `engine/ccl`). *Alternative*: return an error from `GetCol` — rejected, it changes the `Context` interface.
2. **Name-before-index resolution stays as is.** `price * 2` on a table with a column named `price` still resolves `price` as an Excel index; that is the same decision as T-11 (name vs index) and is left for the owner. The change makes the failure loud (out-of-range error) instead of silent zeros, and documents `['name']`.
3. **Duration → seconds** in `toFloat64`, matching `(A - B) / 86400` in the docs. `DAY()` is untouched.
4. **`@` copies** in `dataTableContext.GetCurrentRow`; `MapContext` already built a fresh map.
5. **Nested sequence functions**: `evaluateToColumn` passes a row-independent `[]any` of row length through unchanged. Arithmetic on a sequence result (`LAG(A,1) + A`) stays undefined in v1 as documented.
6. **Panics**: `scalarInt` bounds the value to int32 and rejects NaN/Inf; `callAggregateFunction`/`callSequenceFunction` recover like `callFunction`; `runeSlice` compares instead of adding; `REPEAT` caps at 64 MiB.
7. **Registry**: one `sync.RWMutex`; every read goes through `lookup*`.
8. **Aggregates**: `SUM`, `AVG`, `collectFloats` skip `NaN`. `VAR`/`STDEV`/`VARP`/`STDEVP` keep returning an error on too few values — documented in `CCL.md` and pinned by `TestAggregateStatFunctions`; unifying with `MEDIAN`'s nil is CCL-28 and stays open.
9. **Atomic output**: a shared `writeFileAtomically(path, func(io.Writer) error)`; `ToCSV` is split into `writeCSV(io.Writer)` so the flush error can be tested with a failing writer.
10. **`AtomicDoN` re-entry**: if `holder == gid` for any actor, run `f` inline and fire `TrustZoneFallbackHook` — the same semantics as nested `AtomicDo`. *Alternative*: document "never nest" and keep locking — rejected, a rule nobody can enforce leaves the deadlock in production.
11. **csvxl**: `safeSheetFileName` rejects `/`, `\`, `..`, empty and any name whose `filepath.Base` differs; `saveSheetAsCsv` reads rows first and writes through a temp file.
12. **CLI nil results**: typed `*DataList` locals in `col`/`row`; explicit nil checks in the three transforms. `serializeVariable` also treats a typed nil as a `Raw nil` so an old state cannot crash the next save.
13. **NaN persistence**: `{"$float": "NaN" | "+Inf" | "-Inf"}` markers, applied recursively to `[]any`; DataTable state moves from a JSON-document string to `{columns: [{name, data}], rowNames}` and the reader accepts both.
14. **Root flags**: `wrapRawArgCommand` peels `--env/--no-color/--log-level` off the front of a `DisableFlagParsing` command's args and re-runs `openEnvironment`. *Alternative*: `SetInterspersed(false)` — rejected, `newdl -1 2` would then be parsed as a flag.
15. **Scripts**: `run` sets `OpenREPL = nil` for its duration and counts depth in an unexported `ExecContext` field (limit 16).
16. **History**: `commands.SanitizeHistoryLine` masks the remainder of a `db connect` line (KV `password=` added to `maskDSNPassword`); the REPL uses `DisableAutoSaveHistory` and saves the sanitized line itself; history files are chmod 0600.
17. **Tests**: real assertions for the eleven `// TODO` tests; factor-analysis edge tests assert the current singular-matrix error; the workflow pattern becomes `ScikitLearn`.

## Risks / Trade-offs

- [A CCL user relying on `E` being nil on a short table] → now an error; that behaviour was never documented.
- [`AtomicDoAll` inside `AtomicDo` no longer locks the other instances] → same guarantee as nested `AtomicDo`, documented in AGENTS.md and both docs; the hook still logs it.
- [State-file layout change] → old files still load; new files are not readable by older CLI builds.
- [`REPEAT` cap at 64 MiB] → a legitimate larger string is refused with a clear message.
