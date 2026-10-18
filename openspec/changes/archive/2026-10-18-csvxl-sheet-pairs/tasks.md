## 1. Tests first

- [x] 1.1 csvxl tests for every scenario of `csvxl-sheet-pairing`, the `FileNames` extension rule, names checked before any CSV is written, and the deprecated names keeping their behaviour. The new-name tests fail to compile on the old code.
- [x] 1.2 CLI tests for `convert` pairs, unknown sheet, duplicate file, the formula guard, the unchanged plain form, `sheets`, and completion.

## 2. Implementation

- [x] 2.1 `SheetFromCSV`, the new `CSVToExcel` and `AppendCSVToExcel` with duplicate-sheet refusal, delegating to the deprecated functions.
- [x] 2.2 `ExcelToCSVOptions.FileNames`, the new `ExcelToCSV` planning every file name before writing; `ExcelDirToCSV` refuses `FileNames` and cross-workbook collisions.
- [x] 2.3 `SheetNames`.
- [x] 2.4 CLI `convert` sheet form and `sheets`, with Usage, Forms and Examples.

## 3. Documentation

- [x] 3.1 `Docs/csvxl.md` and `Docs/cli-dsl.md`.
- [x] 3.2 `CHANGELOG.md` and `CHANGELOG_TW.md`: amend the unreleased csvxl rename entry, add the CLI entry.

## 4. Verify

- [x] 4.1 gofmt, `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run`.
