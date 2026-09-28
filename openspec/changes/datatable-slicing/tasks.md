# Tasks: datatable-slicing

## 1. Tests first

- [x] 1.1 `datatable_slice_test.go`: `SliceRows` and `SliceCols` return the range with the table name, column names and row names, the whole table for the full range, and nothing for an empty range; six out-of-range bounds record an error and return an empty table; the results own their data. Against the old code the file does not compile: `SliceRows` is undefined.
- [x] 1.2 Same file: each of the ten deprecated index methods equals its slice for every column letter and row index of the fixture; `Headers` and `SetHeaders` equal `ColNames` and `SetColNames`.
- [x] 1.3 Same file: `FilterColsByColIndexLessThan("Z")` and `…LessThanOrEqualTo("Z")` keep every column. Run on the old code on its own, it panicked with `slice bounds out of range [:25] with capacity 4`.

## 2. Implementation

- [x] 2.1 `datatable_filters.go`: `SliceRows` and `SliceCols`; the two less-than column methods clamp to the last column; the ten index methods Deprecated with their replacement named.
- [x] 2.2 `datatable_colname.go`: `Headers` and `SetHeaders` Deprecated. `interfaces.go`: `IDataTable` lists `SliceRows` and `SliceCols`.

## 3. Docs, changelog, ledger

- [x] 3.1 `Docs/DataTable.md`: a Slicing section with `SliceRows` and `SliceCols` and a line in the table of contents; a Deprecated note on the ten index methods, `Headers` and `SetHeaders`.
- [x] 3.2 Skills: no change. Neither skill names these methods.
- [x] 3.3 `CHANGELOG.md` and `CHANGELOG_TW.md`: Core.
- [x] 3.4 `api-review.md`: T-21 marked partly fixed; `DataTable.Counter` waits for the owner.
- [x] 3.5 `AGENTS.md`: the removal follow-up for `FilterByCustomElement` now covers the ten methods, `Headers` and `SetHeaders`. `delivery-status.md`: a Latest Milestones entry.

## 4. Verification

- [x] 4.1 gofmt, `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run` (0 issues), `openspec validate datatable-slicing --strict`.
