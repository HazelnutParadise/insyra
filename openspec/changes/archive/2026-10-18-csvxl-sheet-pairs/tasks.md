## 1. Tests first

- [x] 1.1 csvxl tests for every scenario of `csvxl-sheet-pairing`, the `FileNames` extension rule, and names checked before any CSV is written. Record that they fail on the old code.
- [x] 1.2 CLI tests for `convert` pairs, the one-sheet rule, unknown sheet, duplicate file, and `sheets`.

## 2. Implementation

- [x] 2.1 `SheetFromCSV`, the new `CSVToExcel` and `AppendCSVToExcel`, duplicate-sheet refusal; deprecated wrappers build the slice from the parallel slices.
- [x] 2.2 `ExcelToCSVOptions.FileNames`, the new `ExcelToCSV`, planning every file name before writing; `ExcelDirToCSV` refuses `FileNames` and cross-workbook collisions; deprecated `ExcelToCsv` keeps `csvNames`.
- [x] 2.3 `SheetNames`.
- [x] 2.4 CLI `convert` and `sheets`, with Usage, Forms and Examples.
- [x] 2.5 Move existing tests and callers to the new signatures.

## 3. Documentation

- [x] 3.1 `Docs/csvxl.md` and `Docs/cli-dsl.md`.
- [x] 3.2 `CHANGELOG.md` and `CHANGELOG_TW.md`, BREAKING marked.
- [x] 3.3 C-11 in `api-review.md`, a milestone in `delivery-status.md`, an `AGENTS.md` follow-up for the `dev` backport.

## 4. Verify

- [x] 4.1 gofmt, `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run`.
