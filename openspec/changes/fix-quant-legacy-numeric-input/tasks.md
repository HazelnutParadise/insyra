# Tasks: fix-quant-legacy-numeric-input

## 1. Tests first

- [ ] 1.1 `quant/performance_test.go`：`SharpeRatio` 遇 nil 格、`MaxDrawdown` 與 `AnnualizedReturn` 遇 `"n/a"`、三者遇 NaN／Inf 各回含標籤與列號的錯誤；nil 輸入回錯不 panic；既有數值測試不改
- [ ] 1.2 `quant/overfitting_test.go`：`DeflatedSharpeRatio` 的 `trialSharpes` 含 NaN 回錯；`PBO` 第 2 欄第 3 列為字串時錯誤含 `column 1` 與 `row 3`

## 2. Implementation

- [ ] 2.1 `quant/performance.go`：三個匯出函式改走 `numericSeries`（標籤 `returns`／`equity`），核心不動
- [ ] 2.2 `quant/overfitting.go`：`DeflatedSharpeRatio` 走 `numericSeries(trialSharpes, "trialSharpes")`；`PBO` 每欄走 `numericSeries(col, fmt.Sprintf("column %d", j))`，長度檢查沿用原訊息

## 3. Docs, changelog, skills, follow-up

- [ ] 3.1 `Docs/quant.md`：Input convention 段落改寫成「所有函式拒絕不可讀值」；Error Handling 的 `SharpeRatio`、`MaxDrawdown`、`AnnualizedReturn`、`DeflatedSharpeRatio`、`PBO` 五列加上不可讀格子條件
- [ ] 3.2 `skills/insyra/SKILL.md` quant 段落：把「新的 quant 輸入必須是有限數值」改成「所有 quant 輸入」
- [ ] 3.3 `CHANGELOG.md` 與 `CHANGELOG_TW.md` `## Unreleased` `### quant` 段末各加一條，依過去 release note 的方式標示 breaking
- [ ] 3.4 `AGENTS.md` Follow-ups「`ToF64Slice` still fabricates zeros」條目：從 Where 移除 `quant/`，What 補一句 quant 已於本變更遷移

## 4. Verification

- [ ] 4.1 `go test ./quant/...` 與 `go test ./...` 全綠；`golangci-lint run` 無新錯誤
- [ ] 4.2 `openspec validate fix-quant-legacy-numeric-input --strict` 通過
