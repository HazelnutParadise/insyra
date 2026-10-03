# Tasks: ccl-dateadd-clamps-month-end

## 1. CCL

- [x] 1.1 Expected values from pandas 2.3.3 (`~/.cache/insyra-crosslang-venv`): `Timestamp(d) + DateOffset(months=n)` or `DateOffset(years=n)` for 13 cases, among them `2024-01-31 +1m → 2024-02-29`, `2024-03-31 -1m → 2024-02-29`, `2024-02-29 +1y → 2025-02-28`, `2024-02-29 +4y → 2028-02-29`, `2024-05-31 10:30:15 +1m → 2024-06-30 10:30:15`
- [x] 1.2 Failing tests first: `internal/ccl/dateadd_month_end_test.go`; on the old code 12 of the 13 month and year cases failed (`2024-03-02`, `2025-03-01`, …) and the time-zone case gave March 2
- [x] 1.3 `addMonths` for the `month` and `year` units
- [x] 1.4 End-to-end through `DataTable.AddColUsingCCL` (`ccl_dates_and_quotes_test.go`), failing on the old code

## 2. Docs, changelog, ledger

- [x] 2.1 `Docs/CCL.md`: the month-end rule at `DATEADD`, and `EDATE` in the Excel table
- [x] 2.2 `CHANGELOG.md` / `CHANGELOG_TW.md`: `Core`, BREAKING
- [x] 2.3 `api-review.md` CCL-33; `delivery-status.md`

## 3. Verification

- [x] 3.1 gofmt, `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run`, `openspec validate ccl-dateadd-clamps-month-end --strict`
