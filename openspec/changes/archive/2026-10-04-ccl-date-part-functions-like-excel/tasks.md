# Tasks: ccl-date-part-functions-like-excel

## 1. CCL

- [x] 1.1 Failing tests first: `internal/ccl/date_part_functions_test.go` replaces `duration_functions_test.go` (parts of a date, time zones, a dropped fraction of a second, agreement with `DATEPART` and `DAYOFMONTH`; durations, duration strings and numbers refused with a `DATEDIFF` hint; what the hint points to). On the old code every date-part case failed with `DAY converts a duration to days and was given a date`, and every duration case returned a value
- [x] 1.2 `datePart` (shared with `DATEPART`) and `datePartFunction` in `stdlib_datetime.go`; the duration converters and `durationIn` removed from `stdlib.go`
- [x] 1.3 `datatable_test.go` computes differences with `DATEDIFF`; `portable_integer_args_test.go` drops the duration-conversion cases; `ccl_dates_and_quotes_test.go` checks `DAY` of a date column end to end

## 2. Docs, changelog, ledger

- [x] 2.1 `Docs/CCL.md`: the date table, the `DAY / HOUR / MINUTE / SECOND` section, the duration notes and examples, the Excel table, `DAYOFMONTH` Deprecated
- [x] 2.2 `CHANGELOG.md` / `CHANGELOG_TW.md`: the `DAY` entry and the `DATEPART` entry describe the final behaviour
- [x] 2.3 `api-review.md` CCL-22; `delivery-status.md`

## 3. Verification

- [x] 3.1 gofmt, `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run`, `openspec validate ccl-date-part-functions-like-excel --strict`
