# Tasks: datalist-quantile

## 1. Tests first

- [x] 1.1 Add `datalist_quantile_test.go`. It covers matching pandas, the three scales agreeing, `NaN`/`nil` being skipped, agreement with `Describe`, out-of-range arguments, text cells, an empty list and the interface. Against the old code it fails to compile, because `Quantile` does not exist. Run against existing methods only, `Quartile(3)` returns 2.25 where 2.5 is expected, `Median` returns 1.5 where 2 is expected, and a text cell gives `Quartile` 2 with no error.
- [x] 1.2 Add `internal/dsl/commands/quantile_test.go` and a `quantile p` row in `error_message_test.go`. The old code fails with `unknown command: quantile`.

## 2. Implementation

- [x] 2.1 `datalist.go`:
  - add `sortedObservations`, `quantileOf` and `Quantile`;
  - rewrite `Quartile`, `IQR` and `Percentile` on top of them;
  - make `Median` skip `NaN`.
- [x] 2.2 `interfaces.go`: `IDataList` gains `Quantile`.
- [x] 2.3 `internal/dsl/commands/stats_dl_extra.go`: add the `quantile` command.
- [x] 2.4 `datalist_summary.go` and `plot/boxplot.go`: compute from the numeric cells only. The full test run caught the box plot through `TestHowCellsThatAreNotNumbersAreDrawn`.

## 3. Docs, changelog, ledger

- [x] 3.1 Docs:
  - `Docs/DataList.md`: add the `Quantile` section, and correct `Quartile`, `IQR`, `Percentile` and `Median`;
  - `Docs/quant.md`: state the `PercentileBands` scale and add the `WalkForward` and `SliceRows` example;
  - `Docs/cli-dsl.md`: add `quantile`.
- [x] 3.2 `CHANGELOG.md`/`CHANGELOG_TW.md`: entries under Core and CLI.
- [x] 3.3 `AGENTS.md`: record the three-name exception and the `Median` follow-up.
- [x] 3.4 `api-review.md` D-13, QU-3, and the issue map.
- [x] 3.5 `delivery-status.md`.
- [x] 3.6 Skills: no change. No principle, workflow or documentation location changed.

## 4. Verification

- [x] 4.1 gofmt, `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run`, `openspec validate datalist-quantile --strict`
