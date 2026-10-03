# Tasks: py-dataframe-name-not-a-column

## 1. 測試先行

- [x] 1.1 `py/environment_setup_test.go`：端對端測試加上有 `name` 欄的 DataFrame 沒有表名、手動設 `df.name` 的 DataFrame 表名照舊。以真的環境在舊程式上確認失敗

## 2. 實作

- [x] 2.1 `py/builtin.go`：pandas DataFrame 分支只在 `name` 不是欄位時讀表名；以真的環境跑過 1.1

## 3. 文件與紀錄

- [x] 3.1 `Docs/py.md`：Supported Return Mappings 說明 `name` 欄不會被當成表名
- [x] 3.2 `CHANGELOG.md`、`CHANGELOG_TW.md`：`### py` 段落末尾新增條目
- [x] 3.3 `AGENTS.md`：刪掉這條 follow-up
- [x] 3.4 `delivery-status.md`：Latest Milestones 最上方新增條目

## 4. 驗證

- [x] 4.1 `gofmt -s -l`、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`（0 issues）、`openspec validate py-dataframe-name-not-a-column --strict`
