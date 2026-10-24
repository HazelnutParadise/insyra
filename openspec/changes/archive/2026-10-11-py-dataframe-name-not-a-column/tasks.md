# Tasks: py-dataframe-name-not-a-column

## 1. 測試先行

- [x] 1.1 `py/dataframe_name_test.go`：假 Python 產生 runner 的 Python，測試有 `name` 欄的 DataFrame 沒有表名、手動設 `df.name` 的 DataFrame 表名照舊。在舊程式上確認失敗

## 2. 實作

- [x] 2.1 `py/builtin.go`：pandas DataFrame 分支在有 `name` 欄時從 frame 自己的 `__dict__` 讀表名，不走 `getattr`；1.1 通過

## 3. 文件與紀錄

- [x] 3.1 `Docs/py.md`：Supported Return Mappings 說明 `name` 欄不會被當成表名
- [x] 3.2 `CHANGELOG.md`、`CHANGELOG_TW.md`：`### py` 段落末尾新增條目

## 4. 驗證

- [x] 4.1 `gofmt -s -l`、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`（0 issues）
