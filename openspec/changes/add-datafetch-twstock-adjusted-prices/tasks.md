# Tasks: add-datafetch-twstock-adjusted-prices

## 1. Tests first

- [ ] 1.1 `twstock_parse_test.go`：`parseROCDate("115年06月01日")` → 2026-06-01；不合法長格式回錯
- [ ] 1.2 `twstock_test.go` `ExRights`：fixture `twse_twt49u_20260601_20260903.json` 首列各欄位與 `AdjFactor`；`Kind` 三種對應；超過一年的區間切成三次請求且首尾相接；缺「除權息參考價」表頭回錯；`from > to` 回錯；`TWMarketTPEx` 在未支援時回含 `not supported` 的錯誤
- [ ] 1.3 `twstock_test.go` `DailyPricesAdjusted`：用合成的 STOCK_DAY 與 TWT49U fixture（測試內建 JSON，不必錄製）驗證單一除息日縮放、兩個除權息日累乘、無除權息日不變、除息日 `AdjClose` 報酬為 0 而 `Close` 報酬為 −2%、nil 價格對應 nil
- [ ] 1.4 `twstock_live_test.go`：加 `ExRights`（TWSE，最近 90 天）與 `DailyPricesAdjusted`（2330，最近 90 天）子測試，仍受 `INSYRA_RUN_LIVE_TWSTOCK` 閘門

## 2. Implementation

- [ ] 2.1 `twstock_parse.go`：`parseROCDate` 支援 `年月日` 長格式；新增 `exRightsHeaderAliases` 與 `Kind` 對應
- [ ] 2.2 `twstock.go` `ExRights`：TWSE 一年分頁、`mapHeaders`、`AdjFactor` 計算、依 `Date` 再 `Code` 排序；TPEx 端點可確認則實作，否則回明確錯誤；`Auto` 只查 TWSE（TPEx 未支援時）
- [ ] 2.3 `twstock.go` `DailyPricesAdjusted`：呼叫 `DailyPrices` 與 `ExRights`，依 `Code` 過濾，除權息日遞減走訪累乘因子，產生 `AdjFactor`、`AdjOpen`、`AdjHigh`、`AdjLow`、`AdjClose`
- [ ] 2.4 若 TPEx 端點確認：錄製 fixture 到 `datafetch/testdata/twstock/` 並補 TPEx 測試；否則在 design.md 的 Open Questions 記下探測結果

## 3. Docs, changelog & skills

- [ ] 3.1 `Docs/datafetch.md` TWStock 章節：新增 `ExRights` 與 `DailyPricesAdjusted` 的方法、欄位表、向後調整慣例、`to` 之後不納入的說明、TPEx 支援狀態、與 Yahoo `Adj Close` 的差異
- [ ] 3.2 `Docs/quant.md`：Beta 的對齊配方改用 `DailyPricesAdjusted` 的 `AdjClose` 作為台股範例，並說明為什麼
- [ ] 3.3 `skills/insyra/SKILL.md` datafetch 段落加一句：算報酬用 `AdjClose`
- [ ] 3.4 `CHANGELOG.md` 與 `CHANGELOG_TW.md` `## Unreleased` `### datafetch` 段末各加一條

## 4. Verification

- [ ] 4.1 `go test ./datafetch/...` 與 `go test ./...` 全綠且預設不觸網；`INSYRA_RUN_LIVE_TWSTOCK=1 go test ./datafetch/ -run TWStockLive` 通過一次；`golangci-lint run` 無新錯誤
- [ ] 4.2 `openspec validate add-datafetch-twstock-adjusted-prices --strict` 通過
