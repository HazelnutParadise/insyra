# Tasks: add-quant-beta-capm

## 1. Tests first (quant/capm_test.go)

- [x] 1.1 一致性測試：隨機報酬序列（固定 seed）上，`CAPM(asset, market, 0)` 的 `Beta`、`Alpha`、`BetaStdErr`、`AlphaStdErr`、`RSquared` 分別與 `stats.LinearRegression(asset, market)` 的 `Slope`、`Intercept`、`StandardError`、`StandardErrorIntercept`、`RSquared` 相等（1e-12），`N` 等於長度；`Beta(asset, market)` 等於 `Slope`
- [x] 1.2 golden 測試：spec 中的六期序列，手算 `Beta`、`Alpha`、`RSquared` 寫死於測試（1e-9），並附手算過程註解
- [x] 1.3 結構測試：`asset = 1.5·market + 0.001` 時 `Beta == 1.5`；常數 `asset` 時 `Beta == 0`、`RSquared` 為 NaN、兩個標準誤為 0、無錯誤
- [x] 1.4 無風險利率測試：`rf = 0` 與 `rf = 0.0002` 的 `Beta` 相等且等於 `Beta(asset, market)`；`Alpha_rf == Alpha_0 − rf·(1−Beta)`
- [x] 1.5 驗證測試：nil 輸入、長度不同、n < 3、`market` 常數、`asset[0] == nil`（模擬未清理的 `PctChange`）、`"n/a"`、NaN、Inf 各回錯；錯誤訊息含條件描述，不可讀格子含序列名與列號

## 2. Implementation (quant/capm.go)

- [x] 2.1 `CAPMResult` 型別與 doc comment（輸入為對齊後的每期報酬、`rf` 每期、`Alpha` 每期、常數 asset 時 `RSquared` 為 NaN）
- [x] 2.2 `capmF64(asset, market []float64, rf float64) (*CAPMResult, error)`：長度、n ≥ 3、benchmark 變異數檢查；閉式 OLS（`Sxx`、`Sxy`、`SSR`、`SST`、`s² = SSR/(n−2)`）
- [x] 2.3 `CAPM` 匯出包裝：先檢查 nil 與長度，再以 `numericSeries(asset, "asset")`、`numericSeries(market, "market")` 轉換，呼叫核心
- [x] 2.4 `Beta` 匯出函式：與 `CAPM` 相同驗證路徑，回傳 `Sxy / Sxx`
- [x] 2.5 更新 `quant/init.go` 套件 doc comment，加上 market beta / CAPM 一句

## 3. Docs, changelog & skills

- [x] 3.1 `Docs/quant.md`：Overview 加 **Market exposure** bullet；在 Performance Metrics 之後新增「Market Exposure (CAPM)」章節（`CAPMResult`、`Beta`、`CAPM`、參數、回傳、錯誤條件、`rf` 與 `Alpha` 的每期慣例、常數 asset 的 NaN）；Usage Examples 加「Beta of a stock against its index」範例，含兩張價格表 `Merge` on date → `PctChangeCol` → `ClearNils` 的對齊配方，以及 `Close` vs `Adj Close`、視窗長度、頻率會改變結果的提醒；Error Handling 加 `Beta` / `CAPM` 兩列；Related Packages 提到 `stats.LinearRegression` 供需要 p 值與信賴區間時使用
- [x] 3.2 `Docs/README.md`（第 94 列與 quick-lookup 表）、`README.md`、`README_TW.md` 的 quant 列加入 market beta / CAPM
- [x] 3.3 `skills/insyra/SKILL.md` quant 段落加 `Beta` / `CAPM` 用法與對齊配方一句，指向 `Docs/quant.md`
- [x] 3.4 `CHANGELOG.md` 與 `CHANGELOG_TW.md` `## Unreleased` 的 `` ### `quant` `` 段末各新增一條

## 4. Verification

- [x] 4.1 `go test ./quant/...` 與 `go test ./...` 全綠；`golangci-lint run` 無新錯誤
- [x] 4.2 `openspec validate add-quant-beta-capm --strict` 通過
