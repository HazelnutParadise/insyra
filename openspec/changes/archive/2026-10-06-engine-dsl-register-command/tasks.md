# Tasks: engine-dsl-register-command

## 1. 測試先行

- [x] 1.1 `engine/dsl/commands_test.go`：只用 `engine/dsl` 註冊命令並在 session 執行、輸出與存檔、參數數量檢查、名稱重複與缺 `Run` 的拒絕、命令錯誤傳回 `Execute`；在舊程式上編譯失敗
- [x] 1.2 `cli/commands/deprecated_names_test.go`：九個名稱有指向 `engine/dsl` 的 Deprecated，其餘沒有；在舊程式上失敗

## 2. 實作

- [x] 2.1 `engine/dsl/commands.go`：九個名稱
- [x] 2.2 `cli/commands/commands.go`：九個名稱標 Deprecated
- [x] 2.3 `cli/root.go`、`cli/repl` 改用 `engine/dsl` 的 `ExecContext`；`engine/dsl/api_test.go` 刻意保留舊寫法

## 3. 文件與紀錄

- [x] 3.1 `Docs/cli-dsl.md`：新增「Registering your own command」，範例在 scratchpad 的獨立 module 實際執行過；欄位表寫明 `Aliases` 只對單次執行有效
- [x] 3.2 `engine/README.md`
- [x] 3.3 `CHANGELOG.md`、`CHANGELOG_TW.md`：`### CLI` 段落末尾新增條目
- [x] 3.4 `AGENTS.md`：follow-up 記錄下一版移除 `cli/commands` 的 Deprecated 名稱
- [x] 3.5 `delivery-status.md`：Latest Milestones 最上方新增條目
- [x] 3.6 skills：不需要修改

## 4. 驗證

- [x] 4.1 `gofmt -s -l`、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`（0 issues）、`openspec validate engine-dsl-register-command --strict`；`go list -deps ./engine/dsl` 沒有 `cli/`、Cobra、readline
