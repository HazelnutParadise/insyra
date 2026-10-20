# Tasks

- [x] Shared column lookup helper used by `QuantileSort` and `FamaMacBeth` (`quant/quantile_sort.go`), `QuantileSort` tests unchanged and passing.
- [x] `FamaMacBeth`, `FamaMacBethOptions`, `FamaMacBethResult` (`quant/fama_macbeth.go`).
- [x] `linearmodels` baseline script and JSON under `quant/testdata`.
- [x] Tests: exact cross-section, linearmodels agreement at lags 0, 3, 5, lag sign translation, skipped thin and collinear days, Standardize, column letters, bad input.
- [x] `Docs/quant.md` section, entries in `CHANGELOG.md` and `CHANGELOG_TW.md` under `## Unreleased`.
- [x] gofmt, `golangci-lint run`, `go test ./quant/...`, `openspec validate --specs --strict`.
