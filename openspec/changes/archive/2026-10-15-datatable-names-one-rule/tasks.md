# Tasks: datatable-names-one-rule

## 1. Tests first

- [x] 1.1 Add `datatable_names_t14_test.go`. All seven of its tests failed on f8912873; the fewer-names test passed there, because that behaviour is unchanged
- [x] 1.2 `TestOutOfRangeArgumentsAreRejected`: add too few and too many `setrownames` names. Both were accepted on f8912873
- [x] 1.3 Change case 2 of `TestSetColNames`, which pinned the column growth that was ruled out, to the new rule
- [x] 1.4 Add `TestSimpleRandomSampleKeepsColumnNames`. Measured: without the sampling fix the sample's names come back empty

## 2. Implementation

- [x] 2.1 Add `uniqueName`. Give `safeColName`/`safeRowName` a function name for the warning, and add `…Except` variants that skip the target itself
- [x] 2.2 `SetColNames` and `SetRowNames`: ignore extra names with a warning, and check names only within the call
- [x] 2.3 `SetColNameByIndex`/`ByNumber`, `ChangeColName`, `SetRowToColNames`, `SetColToRowNames`, `SetRowNameByIndex`, `UpdateRow`: skip the target itself
- [x] 2.4 Sampling keeps the silent suffix (`uniqueRowName`). `SimpleRandomSample` creates named columns
- [x] 2.5 CLI `setrownames` refuses a count mismatch

## 3. Docs, changelog, ledger

- [x] 3.1 `Docs/DataTable.md` (`SetColNames`, `SetRowNames`, `AppendCols`) and `Docs/cli-dsl.md`
- [x] 3.2 `CHANGELOG.md` and `CHANGELOG_TW.md`: Core and CLI, both marked BREAKING
- [x] 3.3 `api-review.md` T-14, `delivery-status.md`, AGENTS.md follow-ups
