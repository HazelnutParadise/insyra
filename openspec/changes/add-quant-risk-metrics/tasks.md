# Tasks: add-quant-risk-metrics

## 1. Tests first (quant/risk_test.go)

- [ ] 1.1 VaR 測試：歷史法對照 `DataList.Percentile` 取負；參數法對照 `−(mean + NormPPF(1−c)·sd)`；CVaR ≥ VaR；`confidence` 為 0、1、1.5 回錯；未知 method 回錯；少於 2 筆回錯；不可讀格子指出列號
- [ ] 1.2 Sortino 測試：spec 的手算案例；全部高於 MAR 回錯；`periodsPerYear <= 0` 回錯
- [ ] 1.3 Calmar 測試：等於 `AnnualizedReturn / MaxDrawdown`；單調 equity 回錯（回撤為零）
- [ ] 1.4 Information ratio 測試：手算案例；`returns == benchmark` 回錯；長度不同回錯
- [ ] 1.5 DrawdownSeries 測試：最大值等於 `MaxDrawdown`；單調序列全 0；running peak 非正處為 nil；空序列回錯

## 2. Implementation (quant/risk.go)

- [ ] 2.1 `VaRMethod` 常數與 doc comment（正損失、confidence 方向、方法差異）
- [ ] 2.2 `valueAtRiskF64`、`conditionalValueAtRiskF64`（歷史法用 `quantileType7`；參數法用 `stats.NormPPF` 與本地 `normPDF`）與匯出包裝
- [ ] 2.3 `sortinoRatioF64`、`calmarRatioF64`（呼叫 `annualizedReturnF64`、`maxDrawdownF64`）、`informationRatioF64` 與匯出包裝
- [ ] 2.4 `drawdownSeriesF64` 與 `DrawdownSeries`（回傳 `*insyra.DataList`）
- [ ] 2.5 更新 `quant/init.go` 套件 doc comment

## 3. Docs, changelog & skills

- [ ] 3.1 `Docs/quant.md`：Overview 加 **Risk metrics** bullet；新增「Risk Metrics」章節（VaR/CVaR 符號與方法、Sortino 定義、Calmar、IR、DrawdownSeries）；Usage Examples 加風險報表範例；Error Handling 加六列
- [ ] 3.2 `Docs/README.md`、`README.md`、`README_TW.md` 的 quant 列加入 VaR/CVaR 與 Sortino/Calmar
- [ ] 3.3 `skills/insyra/SKILL.md` quant 段落加風險指標一句與範例
- [ ] 3.4 `CHANGELOG.md` 與 `CHANGELOG_TW.md` `## Unreleased` `### quant` 段末各加一條

## 4. Verification

- [ ] 4.1 `go test ./quant/...` 與 `go test ./...` 全綠；`golangci-lint run` 無新錯誤
- [ ] 4.2 `openspec validate add-quant-risk-metrics --strict` 通過
