# Tasks: ccl-duration-functions-refuse-dates

## 1. CCL

- [x] 1.1 Failing tests first: `internal/ccl/duration_functions_test.go` (a date string and a `time.Time` reaching each of the four functions; the replacements; durations unchanged). Against the old code `DAY('2024-01-02T06:00:00Z')` returned `[0.25]`, `HOUR(B)` `[6.504166666666666]`, `MINUTE(C)` `[0]`, and `DAY(A)` failed with `unsupported type for DAY: time.Time`
- [x] 1.2 `DAY`, `HOUR`, `MINUTE`, `SECOND` refuse a date and name the replacement, through one builder that keeps each unit's conversion as it was
- [x] 1.3 End-to-end through `DataTable.AddColUsingCCL` (`ccl_dates_and_quotes_test.go`), failing on the old code

## 2. Docs, changelog, ledger

- [x] 2.1 `Docs/CCL.md`: the four functions refuse a date; the "Differences from Excel" table
- [x] 2.2 `skills/insyra/SKILL.md`: CCL is Excel-like, not Excel
- [x] 2.3 `CHANGELOG.md` / `CHANGELOG_TW.md`: `Core`, BREAKING
- [x] 2.4 `api-review.md` CCL-22; `delivery-status.md`

## 3. Verification

- [x] 3.1 gofmt, `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run`, `openspec validate ccl-duration-functions-refuse-dates --strict`
