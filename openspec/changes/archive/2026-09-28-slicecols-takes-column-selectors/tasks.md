# Tasks: slicecols-takes-column-selectors

## 1. Tests first

- [x] 1.1 `datatable_slice_test.go`: `SliceCols` with `(1, 3)`, `("B", "D")`, `(Name("b"), Name("d"))`, `("b", -1)` and `(1, Name("d"))` gives the same columns; `("C", nil)`, `(nil, "C")`, `(nil, nil)` and `(-2, 4)` give the open ranges; `(Name("c"), "C")` is empty. An unknown name, a letter past the last column, `5`, `-5`, `1.5` and `("C", "B")` each record their error. Against the old code the file did not compile: `SliceCols` took ints.
- [x] 1.2 Same file: the letter forms stand in for the deprecated methods, `SliceCols(letter, nil)` for `GreaterThanOrEqualTo` and `SliceCols(nil, letter)` for `LessThan`, for every column of the fixture.

## 2. Implementation

- [x] 2.1 `datatable_filters.go`: `SliceCols(from, to any)` with `sliceColBound`; five deprecation notes give the letter forms. `interfaces.go`: the new signature.

## 3. Docs, changelog, ledger

- [x] 3.1 `Docs/DataTable.md`: the Slicing section and `SliceCols` describe selector bounds, open ends and the errors; the deprecated column methods show the letter forms.
- [x] 3.2 `CHANGELOG.md` and `CHANGELOG_TW.md`: the unreleased `SliceCols` entry corrected in place.
- [x] 3.3 Skills: no change.
- [x] 3.4 `delivery-status.md`: a Latest Milestones entry.

## 4. Verification

- [x] 4.1 gofmt, `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run` (0 issues), `openspec validate slicecols-takes-column-selectors --strict`.
