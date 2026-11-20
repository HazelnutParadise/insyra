# Tasks: py-results-keep-nulls

## 1. 測試先行

- [x] 1.1 `py/environment_setup_test.go`：端對端測試加上 polars 有 null 的整數與 UInt64 欄、NaN 值、list 與 struct 欄，以及 pandas `Int64`、`boolean`、`string` 欄與 `Int64` Series 含 `pd.NA` 的情況

## 2. 實作

- [x] 2.1 `py/builtin.go`：polars DataFrame 逐欄改用 `to_list()`；pandas 擴充型別的欄位與 Series 改用自身的值，`pd.NA` 送成 `None`
- [x] 2.2 以真的環境跑 1.1（polars 1.44.2），並另以 polars 2.0.0 確認同一段 Python 的輸出

## 3. 文件與紀錄

- [x] 3.1 `Docs/py.md`：缺值、整數與 list／struct 儲存格的說明
- [x] 3.2 `CHANGELOG.md`、`CHANGELOG_TW.md`：`### py` 段落末尾新增條目

## 4. 驗證

- [x] 4.1 `gofmt -s -l`、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`、`openspec validate py-results-keep-nulls --strict`
