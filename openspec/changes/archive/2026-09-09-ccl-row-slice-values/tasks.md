# Tasks: ccl-row-slice-values

## 1. Tests first

- [x] 1.1 `internal/ccl/row_slice_test.go` and `ccl_row_slice_test.go`: `A:B`, `LAG(@, 1)` against `LAG(@.#, 1)`, `LAG`/`LEAD` over a column range, the refusals, and the consumers of a range unchanged

## 2. Implementation

- [x] 2.1 `Evaluate` and `EvaluateStatement` resolve a range result; `IsRowDependent` treats a column range of two columns as row-dependent; `evaluateToColumn` expands a range first; sequence functions read a row-shaped argument one row per element

## 3. Docs and changelog

- [x] 3.1 `Docs/CCL.md`
- [x] 3.2 `CHANGELOG.md` and `CHANGELOG_TW.md`: Core

## 4. Verification

- [x] 4.1 gofmt, `go build ./...`, `go vet ./...`, `go test ./...`, `openspec validate ccl-row-slice-values --strict`
