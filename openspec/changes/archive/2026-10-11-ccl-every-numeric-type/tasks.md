# Tasks: ccl-every-numeric-type

## 1. CCL

- [x] 1.1 `toFloat64` and `toBool` read every Go integer type; `ISNA` and `IFNA` read a `float32` NaN (failing tests first)
- [x] 1.2 A row index of any numeric type is read in `evaluateRowAccess`, dropping a fraction as a `float64` index does; a `NaN`, infinite or past-`int32` index is an error (failing tests first)
- [x] 1.3 An end-to-end test through `DataTable.AddColUsingCCL` and through `parquet.FilterWithCCL` on an `int16` column

## 2. Docs and changelog

- [x] 2.1 `Docs/CCL.md`: which Go types are numbers, and the row index rule
- [x] 2.2 `CHANGELOG.md` / `CHANGELOG_TW.md`: `Core`

## 3. Verification

- [x] 3.1 gofmt, `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run`, `openspec validate ccl-every-numeric-type --strict`
