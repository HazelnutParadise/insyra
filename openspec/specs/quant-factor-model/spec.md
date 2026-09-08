# quant-factor-model Specification

## Purpose
`quant` 的多因子歸因：把資產超額報酬（只從資產扣無風險利率）以 OLS 回歸在具名因子欄上，包裝 `stats.LinearRegression`，回傳依因子名索引的曝險、alpha、標準誤、t／p 值、R²、調整 R²、殘差；單因子結果與 `CAPM` 一致，輸入驗證拒絕不可讀值並指出因子欄與列號。

## Requirements
### Requirement: Multi-factor attribution of excess returns

`quant` SHALL 提供 `FactorModel(asset insyra.IDataList, factors insyra.IDataTable, riskFreeRate float64) (*FactorModelResult, error)`，以 OLS 把 `asset − riskFreeRate` 回歸在 `factors` 的每一欄上（因子欄不減 `riskFreeRate`）。`FactorModelResult` SHALL 含 `Alpha`、`AlphaStdErr`、`AlphaTValue`、`AlphaPValue float64`；`FactorNames []string`（依表格欄序）；`Exposures`、`StdErrs`、`TValues`、`PValues []float64`（與 `FactorNames` 同索引）；`RSquared`、`AdjustedRSquared float64`；`N int`；`Residuals []float64`；方法 `Exposure(name string) (float64, bool)`。數值 SHALL 與 `stats.LinearRegression(assetExcess, factorCols...)` 的 `Coefficients`、`StandardErrors`、`TValues`、`PValues`、`RSquared`、`AdjustedRSquared`、`Residuals` 一致。

#### Scenario: Agrees with stats.LinearRegression

- **WHEN** 三個隨機因子與一個隨機資產、`riskFreeRate: 0`
- **THEN** `Alpha` 等於 `Coefficients[0]`，`Exposures[j]` 等於 `Coefficients[j+1]`，標準誤、t 值、p 值、R²、調整 R²、殘差逐一相等（1e-12 內），`N` 等於長度

#### Scenario: One factor agrees with CAPM

- **WHEN** `factors` 只有一欄且該欄已是超額市場報酬、`riskFreeRate: r`
- **THEN** `Exposures[0]`、`Alpha`、`StdErrs[0]`、`AlphaStdErr` 分別等於 `CAPM(asset, marketRaw, r)` 的 `Beta`、`Alpha`、`BetaStdErr`、`AlphaStdErr`（1e-12 內，`marketRaw = market + r`）

#### Scenario: Exposure lookup by name

- **WHEN** `factors` 欄名為 `MKT`、`SMB`、`HML`
- **THEN** `Exposure("SMB")` 回傳第二個係數與 `true`；`Exposure("MOM")` 回傳 `0, false`

#### Scenario: Risk-free rate shifts only alpha

- **WHEN** 同一輸入以 `riskFreeRate: 0` 與 `riskFreeRate: r` 各呼叫一次
- **THEN** `Exposures` 相等，第二次 `Alpha` 等於第一次 `Alpha − r`

### Requirement: Factor model input is validated and never coerced

`FactorModel` SHALL 在下列情況回錯：`asset` 或 `factors` 為 nil；`factors` 沒有欄位；任一因子欄長度與 `asset` 不同；觀察數少於因子數加 2；任一格子非數值、NaN 或 Inf（錯誤 SHALL 指出序列名或因子欄名與列號）。因子共線導致 `stats.LinearRegression` 回錯時 SHALL 原樣回傳該錯誤。轉換 SHALL NOT 使用 `DataList.ToF64Slice`。

#### Scenario: Too few observations for the factor count

- **WHEN** 3 個因子、4 筆觀察
- **THEN** 回傳提及至少需要 5 筆的錯誤

#### Scenario: Unreadable factor cell names the column

- **WHEN** `HML` 欄第 7 列為 `"n/a"`
- **THEN** 錯誤訊息包含 `HML` 與 `row 7`

#### Scenario: Length mismatch is refused

- **WHEN** `asset` 有 60 筆、`SMB` 欄有 59 筆
- **THEN** 回傳提及 `SMB` 與長度不同的錯誤

#### Scenario: Collinear factors surface the regression error

- **WHEN** 兩個因子欄完全相同
- **THEN** 回傳來自回歸的錯誤而不是數值結果

