## MODIFIED Requirements

### Requirement: Multi-factor attribution of excess returns

`quant` SHALL 提供 `FactorModel(asset insyra.IDataList, factors insyra.IDataTable, riskFreeRate float64, opts ...FactorModelOptions) (*FactorModelResult, error)`，以 OLS 把 `asset − riskFreeRate` 回歸在 `factors` 的每一欄上（因子欄不減 `riskFreeRate`）。不給 `opts` 時結果 SHALL 與給零值選項相同，標準誤為一般 OLS 標準誤。

#### Scenario: One factor agrees with CAPM

- **WHEN** `factors` 只有一欄且該欄已是超額市場報酬、`riskFreeRate: r`
- **THEN** `Exposures[0]`、`Alpha`、`StdErrs[0]`、`AlphaStdErr` 分別等於 `CAPM(asset, marketRaw, r)` 的 `Beta`、`Alpha`、`BetaStdErr`、`AlphaStdErr`（1e-12 內，`marketRaw = market + r`）

#### Scenario: Risk-free rate shifts only alpha

- **WHEN** 同一輸入以 `riskFreeRate: 0` 與 `riskFreeRate: r` 各呼叫一次
- **THEN** `Exposures` 相等，第二次 `Alpha` 等於第一次 `Alpha − r`

## ADDED Requirements

### Requirement: The factor model can use robust or HAC standard errors

`FactorModelOptions{CovType stats.CovType; Lag int}` SHALL 把 `CovType` 交給 `stats.LinearRegressionWithOptions`，`StdErrs`、`TValues`、`PValues` 與 alpha 的對應值 SHALL 與該回歸相同。`CovHAC` 的 `Lag` 為 0 時 SHALL 用 `stats.NeweyWestLag(n)`，正數照用，負數 SHALL 回錯。`FactorModelResult` SHALL 記錄 `CovType` 與實際使用的 `Lag`。給超過一個 `FactorModelOptions` SHALL 回錯。

#### Scenario: HAC alpha agrees with the regression

- **WHEN** `FactorModelOptions{CovType: stats.CovHAC, Lag: 3}`
- **THEN** `AlphaStdErr`、`AlphaTValue`、`AlphaPValue` 與 `stats.LinearRegressionWithOptions` 以 `CovHAC`、lag 3 回歸的截距結果逐一相等，`Lag` 為 3

#### Scenario: Automatic lag

- **WHEN** `CovHAC` 且 `Lag: 0`、120 筆觀察
- **THEN** `Lag` 等於 `stats.NeweyWestLag(120)`

#### Scenario: Negative lag is refused

- **WHEN** `CovHAC` 且 `Lag: -1`
- **THEN** 回傳錯誤
