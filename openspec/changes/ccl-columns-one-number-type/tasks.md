# Tasks: ccl-columns-one-number-type

## 1. Core

- [x] 1.1 Failing tests first: `ccl_number_types_test.go`, through `AddColUsingCCL`, `ExecuteCCL` (`NEW` and assignment), `EditColByNameUsingCCL` and `EditColByIndexUsingCCL`: a literal fallback, integers next to fractions, text and booleans, `nil`, an integer past 2^53, NaN, and an `int16` copy. On the old code the fallback, fraction, mixed-text, past-2^53 and NaN cases failed (`0 (int64), want 0 (float64)`)
- [x] 1.2 `unifyCCLNumbers` and its three call sites
- [x] 1.3 Tests that pinned a mixed result move back: `TestCCL_NumericStringComparison`'s string-and-number case is all `float64` again, and `TestDataTable_ExecuteCCL_StdlibPipeline` uses the documented `COALESCE(TONUM(...), 0)` again

## 2. Docs, changelog, ledger

- [x] 2.1 `Docs/CCL.md`: the one-number-type rule; the `0.0` advice removed
- [x] 2.2 `skills/insyra/SKILL.md`
- [x] 2.3 `CHANGELOG.md` / `CHANGELOG_TW.md`: the `ccl-exact-integer-arithmetic` entry says the column is settled instead of telling users to write `0.0`
- [x] 2.4 `api-review.md` CCL-21; `delivery-status.md`

## 3. Verification

- [x] 3.1 gofmt, `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run`, `openspec validate ccl-columns-one-number-type --strict`
