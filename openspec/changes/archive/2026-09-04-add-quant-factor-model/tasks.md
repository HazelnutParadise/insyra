# Tasks: add-quant-factor-model

## 1. Tests first (quant/factor_test.go)

- [x] 1.1 一致性測試：三因子隨機資料逐欄位對照 `stats.LinearRegression`（1e-12），含 `Residuals`、`AdjustedRSquared`、`N`
- [x] 1.2 單因子對照 `CAPM`：`factors` 為 `market − r`，`riskFreeRate: r`，比對 beta、alpha、兩個標準誤
- [x] 1.3 `Exposure(name)` 命中與未命中；`FactorNames` 順序等於表格欄序
- [x] 1.4 `riskFreeRate` 只平移 alpha
- [x] 1.5 驗證測試：nil、零欄、長度不同（訊息含欄名）、n < k+2（訊息含所需筆數）、不可讀格子（含欄名與列號）、完全共線回錯

## 2. Implementation (quant/factor.go)

- [x] 2.1 `FactorModelResult` 與 `Exposure` 方法、doc comment（因子不減 rf、市場因子需自行減 rf）
- [x] 2.2 `FactorModel`：nil／欄數／長度檢查 → `numericSeries` 逐欄轉換（label 為欄名）→ 減 rf → `stats.LinearRegression` → 重新標記
- [x] 2.3 更新 `quant/init.go` 套件 doc comment

## 3. Docs, changelog & skills

- [x] 3.1 `Docs/quant.md`：Overview 加 **Factor models** bullet；新增「Factor Models」章節（型別、函式、因子慣例、與 CAPM 的關係、錯誤條件）；Usage Examples 加三因子範例；Error Handling 加一列
- [x] 3.2 `Docs/README.md`、`README.md`、`README_TW.md` 的 quant 列加入多因子歸因
- [x] 3.3 `skills/insyra/SKILL.md` quant 段落加 `FactorModel` 一句
- [x] 3.4 `CHANGELOG.md` 與 `CHANGELOG_TW.md` `## Unreleased` `### quant` 段末各加一條

## 4. Verification

- [x] 4.1 `go test ./quant/...` 與 `go test ./...` 全綠；`golangci-lint run` 無新錯誤
- [x] 4.2 `openspec validate add-quant-factor-model --strict` 通過
