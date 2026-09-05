# Tasks: add-cli-quant-portfolio

## 1. Tests first (cli/commands/quant_portfolio_test.go)

- [ ] 1.1 `portfolio` 三個目標各對照直接呼叫 `OptimizePortfolio`（1e-12）；`min`／`max`／`rf`／`target` 傳遞；`_stats` 表內容；`as` 與 `$result`
- [ ] 1.2 錯誤：界長度不符、界非數值、`target` 缺值、未知目標、不可行界的函式庫錯誤含前綴；非 DataTable 變數
- [ ] 1.3 `frontier` 對照 `EfficientFrontier`（列數、`ExpectedReturn`、權重欄名）；`points` 非整數或 < 2 回錯；資產名與固定欄名衝突回錯

## 2. Implementation (cli/commands/quant.go)

- [ ] 2.1 `quantForms` 加 `portfolio` 與 `frontier` 兩項（usage、desc）
- [ ] 2.2 `runQuantPortfolio`：目標關鍵字解析、`rf`、`min`／`max` 逗號清單、呼叫、列印、兩個表
- [ ] 2.3 `runQuantFrontier`：`points`、同樣的選項、寬表輸出、欄名衝突檢查

## 3. Docs, changelog & skills

- [ ] 3.1 `Docs/cli-dsl.md`：`quant` 的 Forms 加兩行；Full Command Index 的 `quant` 列更新；Quickstart 加「Portfolio weights from the CLI」（`fetch tw` 兩三檔 → `pctchange` → `merge` → `quant portfolio`）
- [ ] 3.2 `skills/use-insyra-cli/SKILL.md` 與 references 加兩個形式
- [ ] 3.3 `CHANGELOG.md` 與 `CHANGELOG_TW.md` `## Unreleased` `### CLI` 段末各加一條

## 4. Verification

- [ ] 4.1 `go test ./cli/...` 與 `go test ./...` 全綠；`golangci-lint run` 無新錯誤；`go run ./cmd/insyra help quant` 顯示兩個新形式
- [ ] 4.2 `openspec validate add-cli-quant-portfolio --strict` 通過
