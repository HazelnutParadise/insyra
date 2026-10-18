# Tasks: datalist-quantile

## 1. Tests first

- [x] 1.1 `datalist_quantile_test.go`: pandas values, agreement with `Percentile` and `Quartile`, NaN and nil skipped, refusals
- [x] 1.2 `cli/commands/quantile_test.go` and the argument-error table in `error_message_test.go`

## 2. Implementation

- [x] 2.1 `DataList.Quantile` and `IDataList.Quantile`
- [x] 2.2 the `quantile` CLI command

## 3. Docs and changelog

- [x] 3.1 `Docs/DataList.md` and `Docs/cli-dsl.md`
- [x] 3.2 `CHANGELOG.md` and `CHANGELOG_TW.md`: Core and CLI

## 4. Verification

- [x] 4.1 gofmt, `go build ./...`, `go vet ./...`, `go test ./...`, `openspec validate datalist-quantile --strict`
