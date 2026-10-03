# Tasks: cli-exit-ends-script

## 1. 指令

- [x] 1.1 先寫會失敗的測試：腳本在 `exit`、`quit` 行停止並印出 `script ended by exit at line N`；巢狀 `run` 整條停止且最外層回傳 nil；REPL 與腳本之外的 `exit` 回傳包住 `ErrExit` 的錯誤；`Dispatch` 以別名找到指令；`Session.ExecuteFile` 在 `exit` 行停止回傳 nil；在舊程式上記錄失敗
- [x] 1.2 `exit.go` 依情境回傳 `ErrExit` 或說明錯誤；`run.go` 在 `exit` 停止；`Dispatch` 查別名；`repl.go` 改由指令結束 REPL；`api.go` 的 `ExecuteFile` 在 `exit` 停止；1.1 通過

## 2. 驗證

- [x] 2.1 `gofmt`、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`（0 issues）通過
- [x] 2.2 以建好的 CLI（`HOME` 指向暫存目錄）實測：`insyra run` 含 `exit` 與 `quit` 的腳本、巢狀腳本、one-shot `insyra exit` 的訊息與結束碼、REPL 經 stdin 輸入 `quit`

## 3. 文件

- [x] 3.1 `Docs/cli-dsl.md`：Script Mode、CLI `run` 與 `Session.ExecuteFile` 段落、指令索引的 `exit` 列；`exit` 的 `Usage`／`Description`
- [x] 3.2 `CHANGELOG.md` 與 `CHANGELOG_TW.md` 的 `## Unreleased` 在 `### CLI` 末尾新增條目（標 BREAKING：one-shot `insyra exit` 改為結束碼 1）
- [x] 3.3 `api-review.md` 的 CLI-20 標為已修正；`delivery-status.md` 的 Latest Milestones 最上方新增條目
- [x] 3.4 檢查 `skills/use-insyra-cli/`：只教原則，腳本遇錯繼續的原則不變，本變更不需修改
- [x] 3.5 `openspec validate cli-exit-ends-script --strict` 通過
