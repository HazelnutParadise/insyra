# Proposal: test-unpinned-behaviour

## Why

Four review findings — TS-7/TS-8 (#304), TS-9 (#305), TS-10 (#306) and TS-12 (#308) — are all the same shape: behaviour the documentation or the exported surface promises, with nothing in the suite that would fail if it stopped being true.

- **Two documented `DataTable` behaviours have no test.** `Docs/DataTable.md` shows a worked left-join and right-join example, and `datatable_merge_test.go` only ever calls `MergeModeInner` and `MergeModeOuter`. The same page says `SortBy` "uses stable sort to maintain relative order of equal elements"; every existing sort test either has distinct keys or a second config that breaks every tie, so an unstable sort would pass all three.
- **`internal/core` is at 20.1%.** The whole of `Ring` is at 0%, so is every one of `AtomicDo`, `AtomicDoN`, `Close` and `IsClosed`, and so are `BiIndex.Get`, `DeleteByName`, `Has`, `Len`, `IDs`, `Clone` and `Clear`. These are the primitives the actor model and every row-name lookup in the library are built on; the only reason they are exercised at all today is indirectly, through `insyra`'s own tests.
- **`lp` is at 2.2%.** Four of its five functions need `glpsol` on PATH, but `parseGLPKOutputFromFile`, `createAdditionalInfoDataTable`, `extractIterationNodeCounts` and `extractWarnings` are string and file handling that needs no solver at all, and they are what decides what a caller actually receives.
- **`parquet/ccl.go` is at 0%.** The whole CCL bridge — the 17 `parquetContext` methods that expose an Arrow record to the CCL evaluator, plus `FilterWithCCL` and `ApplyCCL` — has never been run by a test. `parquet` as a package is at 21.2%.

## What Changes

Tests only. No library or CLI behaviour changes, so no changelog entry and no docs to update unless a test finds a documentation error.

- `datatable_merge_leftright_test.go` pins the documented left and right joins by value and row order, both on a key column and on row names.
- `datatable_sort_stability_test.go` pins tie order for an ascending sort, a descending sort, a multi-level sort where rows are equal on every configured column, and row names travelling with their rows.
- `internal/core/ring_test.go` covers `Ring` end to end: an empty ring, growth past the initial capacity, wrap-around after a `PopFront`, `DeleteAt` at the front, middle, back and out of range, and `Clear`.
- `internal/core/biindex_more_test.go` covers the untested `BiIndex` methods and the edge cases the existing file skips: an empty name, a negative id, `Set` moving a name off its old id, `Clone` independence and `Clone` on a nil receiver.
- `internal/core/atomic_test.go` covers the actor: serialisation under concurrency, same-actor and cross-actor re-entry, the trust-zone hook, `AtomicDoN`'s canonical ordering and de-duplication, and what `Close` does to each path.
- `lp/parse_test.go` covers the four solver-free functions against fixture output, including the empty and unreadable cases.
- `parquet/ccl_test.go` writes a Parquet file into `t.TempDir()` and runs `FilterWithCCL` and `ApplyCCL` over it, plus direct tests of the `parquetContext` accessors including their out-of-range answers.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `test-suite-integrity`: it already says tests must assert and reference comparisons must run. It gains what has to be pinned: documented behaviour, and the primitives the library is built on.

## Impact

- Seven new test files. No non-test file changes unless a test finds a defect; any defect found is fixed in its own change so the fix is visible on its own.
- `api-review.md` rows TS-7, TS-8, TS-9, TS-10 and TS-12; `delivery-status.md`.
- #307 (gplot) and #309 (the 16 untested packages and the eight below 50%) are deliberately left out. gplot renders through gonum/plot and needs a different kind of test; #309 is a survey across sixteen packages and several of its numbers have already moved — `cli` root, `internal/algorithms`, `isr` and `gplot` all gained tests since the review. Both go in a later change against fresh measurements.
