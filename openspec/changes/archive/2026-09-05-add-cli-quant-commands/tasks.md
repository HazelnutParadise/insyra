# Tasks: add-cli-quant-commands

## 1. Tests first (cli/commands/quant_test.go)

- [x] 1.1 每個形式一個測試，結果對照直接呼叫 `quant` 函式（scalar 1e-12；表格逐欄）；`as` 與未指定 `as` 的行為
- [x] 1.2 錯誤測試：未知形式列出所有形式；缺必填位置引數；非 DataList／DataTable 變數；函式庫錯誤含 `quant <form>:` 前綴與原訊息；VaR 未知方法關鍵字
- [x] 1.3 `quant bs` Hull 範例輸出與 `quant iv` round trip

## 2. Implementation (cli/commands/quant.go)

- [x] 2.1 註冊 `quant`（Usage、Forms 逐形式、Examples）與 `runQuantCommand` 分派
- [x] 2.2 共用解析：`key value` 選項（`rf`、`mar`、`q`）、call|put、historical|parametric、數值位置引數
- [x] 2.3 輸出與儲存：scalar 印 `name=value` 並存 float；`capm`、`bs` 一列表；`factor` 每因子一列加 `<var>_alpha`；`drawdown` 存 DataList

## 3. Docs, changelog & skills

- [x] 3.1 `Docs/cli-dsl.md`：Command Groups 加 **Quant** 行；Full Command Index 加 `quant` 列；Quickstart 加「Risk report from a return series」（`pctchange` → `clean nil` → `quant sharpe/var/sortino`）
- [x] 3.2 `skills/use-insyra-cli/SKILL.md` 與三份 references 同步 `quant` 形式與範例
- [x] 3.3 `CHANGELOG.md` 與 `CHANGELOG_TW.md` `## Unreleased` `### CLI` 段末各加一條

## 4. Verification

- [x] 4.1 `go test ./cli/...` 與 `go test ./...` 全綠；`golangci-lint run` 無新錯誤；`go run ./cmd/insyra help quant` 顯示所有形式
- [x] 4.2 `openspec validate add-cli-quant-commands --strict` 通過
