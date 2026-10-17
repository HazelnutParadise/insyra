# Tasks: applyccl-keeps-every-column-type

## 1. parquet

- [x] 1.1 Runs carry the file's arrays; every stage keeps them in step with the rows and drops a written column's; the writer writes an unwritten column's array unchanged (failing tests first)
- [x] 1.2 A written column keeps `int8`–`int32`, the unsigned widths, `float32`, `Date32` and `Date64` when every value fits, and the builder builds them (failing tests first)

## 2. Docs, changelog, ledger

- [x] 2.1 `Docs/parquet.md`: which types an untouched and a written column keep
- [x] 2.2 `CHANGELOG.md` / `CHANGELOG_TW.md`: `parquet`
- [x] 2.3 `AGENTS.md`: resolve item 3 of the follow-up; `delivery-status.md`

## 3. Verification

- [x] 3.1 gofmt, `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run`, `openspec validate applyccl-keeps-every-column-type --strict`
