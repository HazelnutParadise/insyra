# Tasks: filter-rows-where

## 1. Tests first

- [x] 1.1 `datatable_filter_rows_where_test.go`: `FilterRowsWhere` compares two columns, hands the predicate the row in column order with its name and a `[]byte` cell intact, gives a copy the predicate cannot write through, keeps the columns when nothing passes, and records an error for a nil predicate. Against the old code the file does not compile: `FilterRowsWhere` is undefined.
- [x] 1.2 Same file: `Filter` keeps a row when any cell passes and passes the column letter; `FilterByCustomElement` equals `Filter` on three tables (fixture, nil and slice cells, empty) and four predicates. Both pass on the old code, run on their own before any change: the deprecation rests on that.
- [x] 1.3 Same file: `Filter`, `FilterRows` and `FilterRowsWhere` survive a function that appends a column to the table. Against the old `Filter` and `FilterRows`, and a first cut of `FilterRowsWhere` that ranged over `dt.columns`, each panicked with `index out of range [2] with length 2`.

## 2. Implementation

- [x] 2.1 `datatable_filters.go`: `FilterRowsWhere`; `Filter`, `FilterRows` and `FilterRowsWhere` work from the columns captured when the call began; doc comments on `Filter` and `FilterRows` say they are any-cell tests; `FilterByCustomElement` Deprecated with its replacement.
- [x] 2.2 `interfaces.go`: `IDataTable` lists `FilterRowsWhere`.

## 3. Docs, changelog, ledger

- [x] 3.1 `Docs/DataTable.md`: `Filter` and `FilterRows` described as any-cell tests; `Filter`'s column argument is the letter; the examples that compared a letter with `"age"`, declared the wrong function type or asserted `x.(int)` are corrected; `FilterByCustomElement` marked Deprecated; a `FilterRowsWhere` section.
- [x] 3.2 Skills: no change. Neither skill names a filter method, and `DataTable.md` is already the page they point to for filtering.
- [x] 3.3 `CHANGELOG.md` and `CHANGELOG_TW.md`: Core.
- [x] 3.4 `api-review.md`: T-13 marked fixed.
- [x] 3.5 `AGENTS.md`: a follow-up to remove `FilterByCustomElement`. `delivery-status.md`: a Latest Milestones entry.

## 4. Verification

- [x] 4.1 gofmt, `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run` (0 issues), `openspec validate filter-rows-where --strict`.
