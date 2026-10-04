# Tasks: dsl-outside-cli

## 1. 測試先行

- [x] 1.1 `engine/dsl/layering_test.go`：`go list -deps` 不得出現 `cli/`、Cobra、readline；在舊程式上失敗（列出 `cli/env`、`cli/style`、`cli/commands`、`cli/repl`、cobra 與 readline）
- [x] 1.2 `engine/dsl/api_test.go`：`NewSession`、`Execute`、`ExecuteFile`、`Context` 的型別與原本相同；在舊程式與新程式上都通過

## 2. 搬移

- [x] 2.1 以 `git mv` 搬 `cli/commands`、`cli/env`、`cli/style` 到 `internal/dsl/`，`cli/repl/api.go` 搬成 `internal/dsl/session.go`（`Session`、`NewSession`），分詞器搬成 `internal/dsl/tokenize.go`（`Tokenize`）
- [x] 2.2 內部 registry 拿掉 Cobra；`BuildCobraCommands` 與旗標的註冊、轉交留在 `cli/commands/cobra.go`，改用可替換的 dispatcher
- [x] 2.3 `internal/dsl/env` 拿掉已 Deprecated 的套件層函式，只留在 `cli/env/env.go`
- [x] 2.4 `cli/commands`、`cli/env`、`cli/style` 以別名與包裝函式保留所有匯出名稱；`engine/dsl` 改匯入 `internal/dsl`
- [x] 2.5 `cli/repl/session.go`：`DSLSession`、`NewDSLSession` 標 Deprecated，指向 `engine/dsl`；`cli/repl/session_test.go` 釘住說明與原本行為
- [x] 2.6 測試分家：會驅動 Cobra 的測試搬到 `cli/commands/cobra_test.go`；內部兩個問 Cobra 的旗標測試改查命令宣告的 `Flags`；`docs_sync_test.go` 的相對路徑改成三層

## 3. 文件與紀錄

- [x] 3.1 `Docs/cli-dsl.md`、`engine/README.md`（DSL 章節原本的簽名早已過時，一併修正）
- [x] 3.2 `skills/use-insyra-cli/SKILL.md`：命令原始碼的位置改了，這是文件位置的變動
- [x] 3.3 `cli/AGENTS.md` 搬到 `internal/dsl/AGENTS.md` 並更新路徑；`AGENTS.md` 的架構段落、`CLAUDE.md` 清單與 follow-up 裡的檔案路徑；`ENG.md` 記下依賴方向
- [x] 3.4 `CHANGELOG.md`、`CHANGELOG_TW.md`：`### CLI` 段落末尾新增條目
- [x] 3.5 `api-review.md`：EN-2 標為已修正
- [x] 3.6 `AGENTS.md`：follow-up 記錄下一版移除 `cli/repl` 的 Deprecated 名稱
- [x] 3.7 `delivery-status.md`：Latest Milestones 最上方新增條目

## 4. 驗證

- [x] 4.1 `gofmt -s -l`、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`（0 issues）、`openspec validate dsl-outside-cli --strict`；`go list -deps ./engine/dsl` 沒有 `cli/`、Cobra、readline
