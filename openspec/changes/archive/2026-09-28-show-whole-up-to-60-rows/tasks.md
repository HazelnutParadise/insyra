# Tasks: show-whole-up-to-60-rows

## 1. Tests first

- [x] 1.1 `show_layout_test.go`: `TestShowPrintsUpToSixtyRowsWhole` shows a 60-row and a 61-row fixture through `DataTable.Show`, `DataList.Show`, `DataTable.ShowTypes` and `DataList.ShowTypes`: 60 rows print whole, 61 print rows 0–19 and 56–60 without row 20. Against the old code all four views truncated the 60-row fixture.
- [x] 1.2 Same file: `TestShowMeasuresOnlyTheRowsItPrints` moves to 70 rows with row 40 hidden, since 30 rows now print whole.

## 2. Implementation

- [x] 2.1 `show.go`: the constant `showWholeUpTo = 60` replaces the literal 25 in `tableViewTruncates` and in the DataList `ShowRangeTo` and `ShowTypesRangeTo`; the doc comments of `ShowRange` and `ShowTypesRange` state the rule.

## 3. Docs, changelog, ledger

- [x] 3.1 `Docs/DataTable.md` and `Docs/DataList.md`: the Show and ShowRange descriptions say 60.
- [x] 3.2 `CHANGELOG.md` and `CHANGELOG_TW.md`: a BREAKING (display only) entry in Core; the sentence of the unreleased `show-measures-shown-rows` entry that stated the 25-row rule removed.
- [x] 3.3 Skills: no change.
- [x] 3.4 `api-review.md`: E-8 marked fixed. `delivery-status.md`: a Latest Milestones entry.

## 4. Verification

- [x] 4.1 gofmt, `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run` (0 issues), `openspec validate show-whole-up-to-60-rows --strict`.
