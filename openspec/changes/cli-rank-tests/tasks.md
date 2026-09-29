# Tasks: cli-rank-tests

## 1. 指令

- [x] 1.1 先寫會失敗的測試：`wilcoxon single`、`wilcoxon paired`、`mannwhitney`、`kruskal` 的輸出等於程式庫以同樣資料與對立假設呼叫的結果；省略對立假設等於雙尾；拼錯的對立假設、參數不足與程式庫錯誤都回傳錯誤；`help` 列出各形式；在舊程式上失敗於指令不存在
- [x] 1.2 `cli/commands/nonparam.go` 註冊並實作三個指令；1.1 通過

## 2. 驗證

- [x] 2.1 `gofmt`、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`（0 issues）通過；以建好的 CLI 實際跑過文件裡的每個範例

## 3. 文件

- [x] 3.1 `Docs/cli-dsl.md`：指令索引新增三列，指令分組加入三個指令，新增一段說明
- [x] 3.2 `CHANGELOG.md` 與 `CHANGELOG_TW.md` 的 `## Unreleased` 在 `### CLI` 末尾新增條目
- [x] 3.3 `delivery-status.md` 的 Latest Milestones 最上方新增條目
- [x] 3.4 檢查 `skills/use-insyra-cli/`：不列指令清單，本變更不需修改
- [x] 3.5 `openspec validate cli-rank-tests --strict` 通過
