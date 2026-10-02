# Tasks: ccl-every-numeric-type

## 1. CCL

- [x] 1.1 `toFloat64` and `toBool` read every Go integer type; `ISNA` and `IFNA` read a `float32` NaN (failing tests first)
- [x] 1.2 A row index of any numeric type goes through `wholeIndex`, on the table and on the streaming path (failing tests first)
- [x] 1.3 An end-to-end test through `DataTable.AddColUsingCCL` and through `parquet.FilterWithCCL` on an `int16` column

## 2. Docs, changelog, ledger

- [x] 2.1 `Docs/CCL.md`: which Go types are numbers
- [x] 2.2 `CHANGELOG.md` / `CHANGELOG_TW.md`: `Core`
- [x] 2.3 `AGENTS.md`: delete the follow-up; `delivery-status.md`

## 3. Verification

- [x] 3.1 gofmt, `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run`, `openspec validate ccl-every-numeric-type --strict`
