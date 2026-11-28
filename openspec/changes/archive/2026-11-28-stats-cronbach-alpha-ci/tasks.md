# Tasks: stats-cronbach-alpha-ci

- [x] 1.1 `stats/cronbach_alpha.go`：`CI`、`ConfidenceLevel`、`GuttmanLambda6`，選項 `ConfidenceLevel`
- [x] 1.2 `stats/testdata/crosslang_baseline.R`、`stats/cronbach_alpha_test.go`：與 psych `alpha.ci`、`G6(smc)` 比對，0.9 水準，錯誤的水準
- [x] 1.3 `Docs/stats.md`、`CHANGELOG.md`、`CHANGELOG_TW.md`
- [x] 1.4 `gofmt`、`go vet`、`go test ./...`、`golangci-lint run`、`openspec validate --strict`
- [x] 1.5 cherry-pick 到 dev 並重跑 1.4
