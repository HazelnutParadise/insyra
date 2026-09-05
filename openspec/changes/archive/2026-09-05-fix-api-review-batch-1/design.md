# Design: fix-api-review-batch-1

## Context

Eight independent bug fixes across four packages, grouped because they share one rule the repo already adopted for `stats` on 2026-08-01: a value the library cannot read must fail, not become `0`. The DataList fixes are the largest group. Today `datalist.go` has three ways of reading a cell as a number — `ToFloat64Safe` (reports failure), `conv.ParseF64` (panics), and `ToF64Slice` (returns `0`) — and the bugs come from the last two.

The failure *shape* of each DataList method (nil, empty list, or self with `Err()`) is review item D-3 and is not decided yet. This change keeps every function's existing failure shape and only fixes what it does to the data.

## Goals / Non-Goals

**Goals:**
- No DataList method rewrites any cell before it knows the whole operation can succeed.
- No path in the touched functions turns an unreadable cell into `0`.
- Results on fully numeric input are bit-identical to before; existing tests pass unchanged.
- The three non-DataList bugs are closed with the smallest change that makes the documented behaviour true.

**Non-Goals:**
- Changing which functions return nil vs. self (D-3), the "skip non-numeric" policy of `Sum`/`Mean`/… (D-4 Med), the Err level (K-3), or deprecating the legacy smoothing methods (D-14).
- Touching `ToF64Slice` / `SliceToF64` themselves or their display-path callers.

## Decisions

1. **One numeric-read helper for DataList** — add `datalist_numeric.go` with `numericCells(data []any, allowMissing bool) ([]float64, int, bool)` returning the converted values, the index of the first unreadable cell, and ok. With `allowMissing`, nil and NaN cells convert to NaN and are not failures. Every touched method calls it once, up front, under the actor lock, and writes only if ok. *Alternative considered*: reuse `stats.numericValues` — rejected because `stats` imports `insyra`, not the reverse, and the DataList helper needs the missing-cell mode.

2. **Missing cells (nil / NaN) are not failures for in-place transforms** — `Normalize`, `Standardize`, `ClearOutliers`, `FillNaNWithMean` already skip them in `Min`/`Max`/`Mean`, and `FillWithMean` in `datalist_impute.go` uses the same `isMissing` rule. They pass through unchanged (`ClearOutliers` keeps them; `Difference` emits NaN for a pair with a missing operand). Only a cell that is something else — a string, a bool, a struct — fails the call. *Alternative*: refuse NaN too — rejected because it would make `Normalize` fail on the very data `ReplaceNaNsWith` exists to clean.

3. **`Rank` follows pandas `na_option="keep"`** — nil/NaN get rank NaN and do not consume a rank position; other non-numeric cells fail. The output stays a float64 list of the input length. *Alternative*: rank NaN last (`na_option="bottom"`) — rejected as a silent choice the user did not make.

4. **Smoothing and interpolation require a fully numeric series** — `ExponentialSmoothing`, `DoubleExponentialSmoothing`, and the six interpolations call `numericCells(data, false)`; a nil or NaN cell is a failure too, because a recursive smoother poisons every later value from a NaN and an interpolation grid cannot have a hole. The new `EWM` and `FillByInterpolation` already carry the missing-aware versions of these operations.

5. **`ReplaceLast` mirrors `replaceFirst_notAtomic`** — same two-branch structure (`!isOldValueNaN && v == oldValue` / `isOldValueNaN && isNaN(v)`), iterated backwards. The `_notAtomic` variant in `datalist_notatomic.go` is checked for the same bug and fixed if present.

6. **`stats` moments use the existing helper** — `insyra.SliceToF64(d)` becomes `numericValues(d, "sample")`; the moment functions and their error messages are otherwise untouched.

7. **`ReadJSON_File` is `os.ReadFile` + `ReadJSON([]byte)`** — one decode path, so the `json.Number` handling cannot drift again. A file holding a single object now loads as one row, matching `ReadJSON`.

8. **`AppendCsvToExcel` deletes before creating** — if `GetSheetIndex(name) != -1`, delete the sheet, then `NewSheet`. `excelize` refuses to delete the only sheet, so when the workbook has exactly one sheet and it is the target, a temporary sheet is created first and removed after the new one exists. Workbook handles: `defer f.Close()` in `AppendCsvToExcel` and `ExcelToCsv`; `EachExcelToCsv` moves the per-file body into a closure so each handle closes at the end of its iteration.

9. **`MaxValues` is checked from metadata** — `ReadColumn` opens the reader, sums `RowGroup(i).NumRows()` over the selected row groups (all when none selected), and returns an error naming the count and the limit before calling `Read`. This honours the documented purpose (avoid loading what is too big) instead of loading and then refusing. *Alternative*: count after reading — rejected as pointless for a memory guard.

## Risks / Trade-offs

- [Callers relied on `Rank` ranking strings as 0] → they now get `Err()` and nil; noted as breaking in the changelog. The review found no such caller in the repo.
- [`ClearOutliers` previously panicked-and-recovered on nil cells, effectively skipping the rest of the loop] → now nil cells are kept and all numeric cells are checked; a list with nils gets *more* outliers removed than before. This is the documented behaviour finally happening; called out in the changelog line.
- [`AppendCsvToExcel` on a single-sheet workbook targeting that sheet] → handled by the temporary-sheet path; covered by a test.
- [Parquet metadata row counts diverge from actual data] → impossible for a valid file; the check is a guard, and `Read` still errors on a corrupt file.

## Open Questions

None. The failure-shape question (D-3) is deliberately left for its own change.
