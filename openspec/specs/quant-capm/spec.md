# quant-capm Specification

## Purpose
`quant` 的市場曝險分析：以已對齊的每期報酬序列計算資產對 benchmark 的市場 beta（`Cov/Var`，等於單一解釋變數 OLS 斜率），以及超額報酬上的單一指數 CAPM 回歸（beta、每期 alpha、R²、標準誤、觀察數）。不做日期對齊與價格轉報酬；輸入驗證採拒絕而非補零。

## Requirements
### Requirement: Market beta of a return series against a benchmark

`quant` SHALL 提供 `Beta(asset, market insyra.IDataList) (float64, error)`，回傳 `Cov(asset, market) / Var(market)`，其中 `asset` 與 `market` 為已按期間對齊的每期報酬序列。共變異數與變異數 SHALL 使用相同的分母（樣本 n−1），使結果等於以 `market` 為唯一解釋變數、對 `asset` 做 OLS 的斜率。`Beta` SHALL 不做日期對齊、不把價格轉成報酬、不丟棄任何格子。

#### Scenario: Beta equals the OLS slope

- **WHEN** 以任意兩條等長、有限、`market` 變異數非零的報酬序列呼叫 `Beta`
- **THEN** 結果等於 `stats.LinearRegression(asset, market).Slope`（浮點容差 1e-12 內）

#### Scenario: A scaled copy of the market has that scale as its beta

- **WHEN** `asset[i] == 1.5 * market[i] + 0.001` 對所有 `i`
- **THEN** `Beta` 回傳 1.5（浮點容差內）

#### Scenario: A constant asset has beta zero

- **WHEN** `asset` 每期報酬都相同、`market` 變異數非零
- **THEN** `Beta` 回傳 0 且無錯誤

### Requirement: Single-index CAPM regression

`quant` SHALL 提供 `CAPM(asset, market insyra.IDataList, riskFreeRate float64) (*CAPMResult, error)`，以 OLS 把超額報酬 `asset − riskFreeRate` 回歸在 `market − riskFreeRate` 上。`riskFreeRate` SHALL 為每期利率，與 `SharpeRatio` 相同慣例。`CAPMResult` SHALL 包含：`Beta float64`（斜率）、`Alpha float64`（截距，每期）、`RSquared float64`、`BetaStdErr float64`、`AlphaStdErr float64`、`N int`（觀察數）。標準誤 SHALL 使用殘差自由度 `N−2`。`asset` 超額報酬變異數為零時，`RSquared` SHALL 為 `NaN`（未定義），`BetaStdErr` 與 `AlphaStdErr` SHALL 為 0，不回錯。

#### Scenario: CAPM agrees with the general linear regression

- **WHEN** 以 `riskFreeRate: 0` 對任意合法輸入呼叫 `CAPM`
- **THEN** `Beta`、`Alpha`、`BetaStdErr`、`AlphaStdErr`、`RSquared` 分別等於 `stats.LinearRegression(asset, market)` 的 `Slope`、`Intercept`、`StandardError`、`StandardErrorIntercept`、`RSquared`（浮點容差 1e-12 內），`N` 等於序列長度

#### Scenario: Beta is invariant to the risk-free rate

- **WHEN** 對同一組輸入分別以 `riskFreeRate: 0` 與 `riskFreeRate: 0.0002` 呼叫 `CAPM`
- **THEN** 兩次的 `Beta` 相等，且都等於 `Beta(asset, market)`

#### Scenario: Alpha shifts with the risk-free rate

- **WHEN** 對同一組輸入分別以 `riskFreeRate: 0` 與 `riskFreeRate: r` 呼叫 `CAPM`
- **THEN** 第二次的 `Alpha` 等於第一次的 `Alpha − r · (1 − Beta)`（浮點容差內）

#### Scenario: Hand-computed golden values

- **WHEN** 以固定的六期報酬對 `asset = {0.010, −0.004, 0.012, 0.003, −0.008, 0.006}`、`market = {0.006, −0.002, 0.008, 0.001, −0.005, 0.004}`、`riskFreeRate: 0` 呼叫 `CAPM`
- **THEN** `Beta`、`Alpha`、`RSquared` 等於以手算 OLS 公式得出並寫死在測試中的值（容差 1e-9），`N == 6`

#### Scenario: A constant asset reports undefined R²

- **WHEN** `asset` 超額報酬每期相同、`market` 變異數非零
- **THEN** `CAPM` 回傳 `Beta == 0`、`Alpha` 等於該常數、`RSquared` 為 `NaN`、兩個標準誤為 0，且無錯誤

### Requirement: Input is validated and never coerced

`Beta` 與 `CAPM` SHALL 在下列情況回傳錯誤且不產生數值：任一輸入為 nil；兩序列長度不同；觀察數少於 3；`market` 變異數為零（beta 未定義）；任一格子非數值、`NaN` 或 `Inf`。錯誤訊息 SHALL 指出是哪個條件；格子不可讀時 SHALL 指出序列名稱（`asset` 或 `market`）與列號（從 1 起算）。轉換 SHALL 經由拒絕不可讀值的路徑，SHALL NOT 使用 `DataList.ToF64Slice`。

#### Scenario: Length mismatch is refused

- **WHEN** `asset` 有 10 筆、`market` 有 9 筆
- **THEN** 兩個函式都回傳提及長度不同的錯誤

#### Scenario: Too few observations are refused

- **WHEN** 兩序列各只有 2 筆
- **THEN** 兩個函式都回傳提及至少需要 3 筆的錯誤

#### Scenario: A flat benchmark is refused

- **WHEN** `market` 每期報酬相同
- **THEN** 兩個函式都回傳提及 benchmark 變異數為零的錯誤

#### Scenario: Unreadable cells are refused with the row named

- **WHEN** `asset` 第 1 格為 `nil`（`PctChange` 未清理的開頭），或任一格為 `"n/a"`、`NaN`、`Inf`
- **THEN** 錯誤訊息包含 `asset` 與 `row 1`（或對應列號），不以 0 代入

#### Scenario: Nil input is refused

- **WHEN** `asset` 或 `market` 為 nil
- **THEN** 回傳指出哪個輸入為 nil 的錯誤

