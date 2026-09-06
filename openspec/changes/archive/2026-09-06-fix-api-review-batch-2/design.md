# Design: fix-api-review-batch-2

## Context

Twenty verified bugs across six packages, all fixable inside existing signatures. The batch deliberately stops at the line where a fix would change an API shape: `Mean() any` keeps returning `any`, `ToJSON_Bytes` still returns nil on a marshal error, and `LogFatal` call sites are untouched — those are review items K-1, D-3, T-12 awaiting a decision.

## Goals / Non-Goals

**Goals:** no panic from a public DataTable method on bad input; no method hands out or shares internal storage; documented semantics hold for the listed methods; hypothesis tests never silently miscount; deterministic output where a map iteration order leaked into results.

**Non-Goals:** the "skip non-numeric" policy of `Sum`/`Mean` (D-4 Med), `Err()` noise (D-5), failure return shapes (D-3), name-vs-index resolution (T-11), any signature change.

## Decisions

1. **Filter results clone via one helper** — `cloneColumnsFor(dt)` deep-copies the selected `*DataList`s (data slice and name) and `dt.rowNames.Clone()`; every `FilterColsBy*`, `FilterCols`, `FilterColsByColNameContains` goes through it. A not-found result is `NewDataTable()` so `rowNames` is never nil. *Alternative*: copy-on-write — rejected, DataList has no such mechanism and the actor model would need to know about sharing.
2. **`Data()` copies per column** — `slices.Clone(col.data)` into the map; `ToMap` delegates. Cost is O(cells), same as `DataList.Data()` already pays.
3. **`DropRowsByIndex` normalises first** — map each index through `normalizeRowIndex`, drop out-of-range, de-duplicate with a set, sort descending, delete. Row names are reindexed after each deletion as today.
4. **`Transpose` iterates rows for names** — the new table's column *i* gets the old row name *i* for every *i* < old row count, independent of the old column count.
5. **`ChangeRowName` uses `safeRowName`** — matching `SetRowNameByIndex`; a collision yields `name_1` rather than stealing the other row's name. *Alternative*: refuse with `Err()` — rejected for consistency with the other setters (D-5 will revisit warnings as a whole).
6. **`AppendRowsByColIndex` grows to the target** — when the parsed index is beyond the current width, append empty columns up to and including it before writing, so `{"Z": 42}` produces column Z holding 42.
7. **One numeric test** — `DropColsContainNumber`/`DropRowsContainNumber` use `IsNumeric`; `Mean` counts cells `ToFloat64Safe` accepts and divides by that count (NaN when zero).
8. **CSV time format** — `time.Time` cells are written with `Format(time.RFC3339Nano)`; the default `ParseDates` layouts already include it. Other types keep `%v`.
9. **CCL methods use named returns** — `(result *DataTable)` set to `dt` before the body runs; the recover path leaves it. The unused `resultDtChan` is removed.
10. **Tests read through `numericSlice`** — replacing `dl.Len()`/`Mean()`/`Stdev()` with values from `numericSlice(dl, label)` and `stat.Mean`/`stat.StdDev`… no: to keep results bit-identical, compute mean and sample stdev with the same two-pass formulas `DataList.Mean`/`Var` use (sum/n; Σ(x−mean)²/(n−1)), on the validated slice. Labels: `data`, `data1`, `data2`, `group <i>`. `CalculateMoment` reads via `numericSlice(dl, "data")`. The z-test effect size stays absolute: the R-verified reference tests pin |d|, so the t/z asymmetry (ST-8) is left as a documented difference rather than changed against the reference.
11. **RFM amount** — `ToFloat64Safe`; a failure warns with the row and skips that row, matching how an unparseable date is already handled in the same loop. Output rows are appended in sorted customer-ID order for `RFM` and `CAI`.
12. **Env name validation** — `ResolveEnvPath` rejects names that are not `^[A-Za-z0-9][A-Za-z0-9._-]*$` or contain `..`; the error names the rule. `Default` and the `default` environment pass.
13. **Deterministic/atomic output** — `createAdditionalInfoDataTable` iterates a fixed key order; `fileGeocodeCache.persist` writes `path+".tmp"` then `os.Rename`.

## Risks / Trade-offs

- [Callers relied on `Filter*` sharing storage to mutate the source through the result] → unlikely and undocumented; changelog marks it breaking.
- [`ChangeRowName` renaming to `x_1` surprises a caller expecting a refusal] → same behaviour as every other setter; noted.
- [Hypothesis tests now error on data with blanks that previously "worked"] → the previous number was wrong; changelog says so and points at `ClearNils`.
