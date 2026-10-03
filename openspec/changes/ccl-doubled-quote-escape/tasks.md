# Tasks: ccl-doubled-quote-escape

## 1. CCL

- [x] 1.1 Check that a doubled quote never compiled: `'it''s'`, `"say ""hi"""`, `''''`, `CONCAT('a''b', 'c')` and `['it''s']` all failed on the old code, and no grammar rule lets a string follow a string
- [x] 1.2 Failing tests first: `internal/ccl/string_escape_test.go` (literals, bracketed column names, statement splitting)
- [x] 1.3 `scanQuoted` reads string literals and bracketed column names
- [x] 1.4 End-to-end through `DataTable.AddColUsingCCL` (`ccl_dates_and_quotes_test.go`), failing on the old code

## 2. Docs, changelog, ledger

- [x] 2.1 `Docs/CCL.md`: single and double quotes, the doubled quote, bracketed names
- [x] 2.2 `CHANGELOG.md` / `CHANGELOG_TW.md`: `Core`
- [x] 2.3 `api-review.md` CCL-27; `delivery-status.md`

## 3. Verification

- [x] 3.1 gofmt, `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run`, `openspec validate ccl-doubled-quote-escape --strict`
