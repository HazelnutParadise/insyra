# Tasks: show-measures-shown-rows

## 1. Tests first

- [x] 1.1 `show_layout_test.go`: `ShowRange(5)`, `Show` on 30 rows and `ShowTypesRange(3)` give the same output for two tables that differ only in a row they do not print. All three failed against the old code.
- [x] 1.2 Same file: a `[]Showable` holding a table and a list goes through `Show`. Against the old code it does not compile.
- [x] 1.3 Same file: `BenchmarkShowRangeFiveOfAMillionRows`, `BenchmarkShowOfAMillionRows`, `BenchmarkShowTypesRangeFiveOfAMillionRows`. Before: 279–437 ms / 5,999,710 allocs, 497–553 ms / 8,000,395 allocs, 173–181 ms / 1,999,955 allocs per call. After: 38–42 µs / 257 allocs, 296–321 ms / 2,001,034 allocs, 18–20 ms / 218 allocs.
- [x] 1.4 A throwaway comparison of `ShowTo`, `ShowRangeTo` with four ranges, `ShowTypesTo` and `ShowTypesRangeTo` on a six-row table with nil, NaN, wide-character, time, bool and row-name cells and on a 30-row table: every full view is byte-for-byte the old output; only partial views changed, by fitting the rows they print.

## 2. Implementation

- [x] 2.1 `show.go`: `tableViewTruncates` and `shownTableRows` decide the printed rows once; `prepareTableLayout`, `prepareTableLayoutTypes` and the new `shownRowLabels` read only those rows; `printRowsColored` and `printTypeRows` take the labels as a map; `typeLabel` holds the type description the layout measures.
- [x] 2.2 `show.go`: `showable` becomes the exported `Showable`.

## 3. Docs, changelog, ledger

- [x] 3.1 `Docs/DataTable.md`: `Show`, `ShowRange` and `ShowTypesRange` say which rows are printed and that widths come from them; `ShowRange()` no longer claims to show every row. `Docs/DataList.md`: the same correction for the list. `Docs/utils.md`: `Show` takes `Showable`, with an example.
- [x] 3.2 Skills: no change.
- [x] 3.3 `CHANGELOG.md` and `CHANGELOG_TW.md`: Core.
- [x] 3.4 `api-review.md`: IN-22 fixed; E-8 partly fixed (`Showable`), with the default truncation left to the owner and the range arguments already settled by `core-settings-batch`.
- [x] 3.5 `delivery-status.md`: a Latest Milestones entry.

## 4. Verification

- [x] 4.1 gofmt, `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run` (0 issues), `openspec validate show-measures-shown-rows --strict`.
