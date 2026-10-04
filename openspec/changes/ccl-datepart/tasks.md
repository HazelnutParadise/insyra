# Tasks: ccl-datepart

## 1. CCL

- [x] 1.1 Failing tests first: `internal/ccl/datepart_test.go` (each unit, case and plural, the date's own time zone, a dropped fraction of a second, errors); `duration_functions_test.go` expects the errors to name `DATEPART`. On the old code every `DATEPART` call failed with `undefined function: DATEPART`
- [x] 1.2 `DATEPART` in `stdlib_datetime.go`; the hints in `durationIn`

## 2. Docs, changelog, ledger

- [x] 2.1 `Docs/CCL.md`: `DATEPART` in the date table, the duration functions' errors, the Excel table
- [x] 2.2 `CHANGELOG.md` / `CHANGELOG_TW.md`: `Core`, and the hint in the `DAY`/`HOUR` entry
- [x] 2.3 `api-review.md` CCL-22; `delivery-status.md`

## 3. Verification

- [x] 3.1 gofmt, `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run`, `openspec validate ccl-datepart --strict`
