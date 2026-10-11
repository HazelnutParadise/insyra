# Tasks: ccl-datepart

## 1. CCL

- [x] 1.1 Failing tests first: `internal/ccl/datepart_test.go` (each unit, case and plural, the date's own time zone, a dropped fraction of a second, errors). On the old code every `DATEPART` call failed with `undefined function: DATEPART`
- [x] 1.2 `DATEPART` in `stdlib_datetime.go`

## 2. Docs and changelog

- [x] 2.1 `Docs/CCL.md`: `DATEPART` in the date table, the Excel table
- [x] 2.2 `CHANGELOG.md` / `CHANGELOG_TW.md`: `Core`

## 3. Verification

- [x] 3.1 gofmt, `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run`, `openspec validate ccl-datepart --strict`
