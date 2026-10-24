# Tasks: show-measures-shown-rows

## 1. Tests first

- [x] 1.1 `show_layout_test.go`: `ShowRange(5)`, `Show` on 30 rows and `ShowTypesRange(3)` give the same output for two tables that differ only in a row they do not print.
- [x] 1.2 Same file: a `[]Showable` holding a table and a list goes through `Show`.
- [x] 1.3 Same file: `BenchmarkShowRangeFiveOfAMillionRows`, `BenchmarkShowOfAMillionRows`, `BenchmarkShowTypesRangeFiveOfAMillionRows`.
- [x] 1.4 對照測試：`ShowTo`、`ShowRangeTo`、`ShowTypesTo` 與 `ShowTypesRangeTo` 完整視圖與原先輸出完全一致，部分視圖只量印出的列。

## 2. Implementation

- [x] 2.1 `show.go`: `tableViewTruncates`（超過 25 列印前 20 後 5）與 `shownTableRows` 決定要印的列；`prepareTableLayout`、`prepareTableLayoutTypes` 與 `shownRowLabels` 只量這些列；`printRowsColored` 與 `printTypeRows` 接收標籤 map。
- [x] 2.2 `show.go`: `showable` 匯出為 `Showable`。

## 3. Docs, changelog, ledger

- [x] 3.1 `Docs/DataTable.md`、`Docs/DataList.md`、`Docs/utils.md`。
- [x] 3.2 `CHANGELOG.md` 與 `CHANGELOG_TW.md`。
- [x] 3.3 `api-review.md`、`delivery-status.md`。

## 4. Verification

- [x] 4.1 全套驗證。
