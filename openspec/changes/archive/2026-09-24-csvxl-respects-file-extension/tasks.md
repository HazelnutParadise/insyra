# Tasks: csvxl-respects-file-extension

## 1. Tests first

- [x] 1.1 `csvxl/file_extension_test.go`: names without `.csv`, `.csv` added when the path is missing, the path read as written when both exist, a directory of the same name, and the append form

## 2. Implementation

- [x] 2.1 `resolveCsvSource` in `csvxl/convert.go`, used by `CsvToExcel` and `AppendCsvToExcel`

## 3. Docs and changelog

- [x] 3.1 `Docs/csvxl.md`
- [x] 3.2 `CHANGELOG.md` and `CHANGELOG_TW.md`: Core

## 4. Verification

- [x] 4.1 gofmt, `go build ./...`, `go vet ./...`, `go test ./...`, `openspec validate csvxl-respects-file-extension --strict`
