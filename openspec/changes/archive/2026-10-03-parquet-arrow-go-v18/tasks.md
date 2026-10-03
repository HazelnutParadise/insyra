# Tasks: parquet-arrow-go-v18

## 1. Dependency

- [x] 1.1 `parquet` imports `github.com/apache/arrow-go/v18` v18.8.0; `go.mod` and `go.sum` tidied; every Deprecated use moved to its replacement
- [x] 1.2 An error the Arrow reader returns for a row group names the file and the row groups that did not read in full and wraps the reader's error; a context error is returned as it is (failing tests first)
- [x] 1.3 A damaged Snappy page is an error on every reader, and `ApplyCCL` leaves the file (test that ends the test binary on v17)

## 2. Docs, changelog, ledger

- [x] 2.1 `Docs/parquet.md`: the damaged-file error
- [x] 2.2 `CHANGELOG.md` / `CHANGELOG_TW.md`: `parquet`
- [x] 2.3 `AGENTS.md`: resolve the follow-ups on moving to v18 and on the damaged Snappy page; `delivery-status.md`

## 3. Verification

- [x] 3.1 gofmt, `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run`, `govulncheck ./...`, `openspec validate parquet-arrow-go-v18 --strict`
