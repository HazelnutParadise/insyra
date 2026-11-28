# Tasks: stats-cronbach-alpha

## 1. 實作

- [x] 1.1 `stats/cronbach_alpha.go`：`CronbachAlpha`、`CronbachAlphaOptions`、`CronbachAlphaResult`、`CronbachAlphaItem`、`String`、`Show`

## 2. 測試

- [x] 2.1 `stats/testdata/crosslang_baseline.R`：`cronbach_alpha` 方法，呼叫 `psych::alpha`
- [x] 2.2 `stats/cronbach_alpha_test.go`：與 psych 比對（六題、兩題、負相關題、反向題）、逐列刪除、三種選欄寫法、反向題的平均數、各種錯誤

## 3. 文件

- [x] 3.1 `Docs/stats.md`、`CHANGELOG.md`、`CHANGELOG_TW.md`

## 4. 驗證與移植

- [x] 4.1 `gofmt -s -l`、`go vet ./...`、`go test ./...`、`golangci-lint run`、`openspec validate --strict`
- [x] 4.2 cherry-pick 到 dev 並重跑 4.1
