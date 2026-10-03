# Tasks: ccl-exact-integer-arithmetic

## 1. CCL

- [x] 1.1 Failing tests first: `internal/ccl/integer_arithmetic_test.go` (literals, columns, overflow, `#`, `SUM`/`MIN`/`MAX`/`MOD`, streaming forms, `TOSTR`) and `ccl_integer_arithmetic_test.go` through `DataTable.AddColUsingCCL`. On the old code 59 internal cases and 16 table cases failed, among them `A + 0` on `int64(9007199254740993)` giving `9.007199254740992e+15`
- [x] 1.2 Integer literals (`cclIntegerNode`), negative literals read with their sign so `-9223372036854775808` is an `int64`
- [x] 1.3 `applyOperator`: two integers through `int64` with overflow checks; `nil` next to an integer is an integer 0; `#` is an `int64`; folded `int64` aggregates become integer literals
- [x] 1.4 `SUM`, `MIN`, `MAX` and their streaming forms share `integerAggregate`; `MOD` follows `%`; `TOSTR` formats an integer under a float verb as a `float64`
- [x] 1.5 Tests that pinned `float64` results of integer arithmetic move to `int64`, each checked to be a type-only change: root CCL tests, `engine/ccl`, `isr`, `internal/ccl` fold, depth, global-row and streaming tests, and `parquet`'s narrow-type and whole-file tests (an integer written into a float64 file column is stored as the float64 it equals)

## 2. Docs, changelog, ledger

- [x] 2.1 `Docs/CCL.md`: literals, the integer rules, overflow, `#`, `SUM`/`MIN`/`MAX`/`MOD`/`TOSTR`, nil, the coercion table, the Excel table; `Docs/parquet.md`: the column type a statement's integers give
- [x] 2.2 `skills/insyra/SKILL.md`: integers stay integers, write `0.0` for a float
- [x] 2.3 `CHANGELOG.md` / `CHANGELOG_TW.md`: `Core`, BREAKING
- [x] 2.4 `api-review.md` CCL-21; `delivery-status.md`

## 3. Verification

- [x] 3.1 gofmt, `go build ./...`, `go vet ./...`, `go test ./...`, `GOARCH=amd64 go test ./internal/ccl`, `golangci-lint run`, `openspec validate ccl-exact-integer-arithmetic --strict`
