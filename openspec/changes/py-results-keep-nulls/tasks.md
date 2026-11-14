# Tasks: py-results-keep-nulls

## 1. 測試先行

- [x] 1.1 `py/missing_values_e2e_test.go`：以 `INSYRA_PY_E2E=1` 執行的測試，涵蓋 polars 有 null 的整數欄、NaN 值、list 與 struct 欄，以及 pandas `Int64`、`boolean`、`string` 欄與 `Int64` Series 含 `pd.NA` 的情況

## 2. 實作

- [x] 2.1 `py/builtin.go`：pandas 與 polars DataFrame 改為逐欄送出，polars 欄位用 `to_list()`；pandas 擴充型別的欄位與 Series 用自身的值，`pd.NA` 送成 `None`
- [ ] 2.2 以真的環境跑 1.1

## 3. 文件與紀錄

- [x] 3.1 `Docs/py.md`：缺值與 list／struct 儲存格的說明
- [x] 3.2 `CHANGELOG.md`、`CHANGELOG_TW.md`：Unreleased 新增 `### py` 段落與條目

## 4. 驗證

- [ ] 4.1 `gofmt -s -l`、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`、`openspec validate py-results-keep-nulls --strict`
