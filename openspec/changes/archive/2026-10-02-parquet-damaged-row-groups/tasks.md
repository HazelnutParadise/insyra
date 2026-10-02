# Tasks: parquet-damaged-row-groups

## 1. parquet

- [x] 1.1 `readTableFrom` and `streamAsArrowRecord` compare the rows read with the metadata's count for the selected row groups and fail with an error naming the file, both counts and the row groups that did not read in full (failing tests first)

## 2. Docs, changelog, ledger

- [x] 2.1 `Docs/parquet.md`: what a damaged row group does
- [x] 2.2 `CHANGELOG.md` / `CHANGELOG_TW.md`: `parquet`
- [x] 2.3 `AGENTS.md`: resolve the follow-up and record the upstream report for the owner; `delivery-status.md`

## 3. Verification

- [x] 3.1 gofmt, `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run`, `openspec validate parquet-damaged-row-groups --strict`
