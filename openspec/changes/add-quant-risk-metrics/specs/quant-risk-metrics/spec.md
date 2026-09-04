## ADDED Requirements

### Requirement: Value at risk and conditional value at risk

`quant` SHALL 提供 `VaRMethod`（`VaRHistorical`、`VaRParametric`）、`ValueAtRisk(returns insyra.IDataList, confidence float64, method VaRMethod) (float64, error)` 與 `ConditionalValueAtRisk(returns insyra.IDataList, confidence float64, method VaRMethod) (float64, error)`。兩者 SHALL 以正的損失比例回報。`VaRHistorical` SHALL 取報酬的 R type-7 分位數 `q = quantile(1 − confidence)` 並回傳 `−q`；其 CVaR SHALL 為所有 `r <= q` 的報酬平均之負值。`VaRParametric` SHALL 為 `−(mean + z·sd)`，`z = NormPPF(1 − confidence)`，`sd` 為樣本標準差；其 CVaR SHALL 為 `−(mean − sd·φ(z)/(1 − confidence))`。`confidence` 不在 `(0, 1)` 內、少於 2 筆、未知 `method`、不可讀／NaN／Inf 的格子 SHALL 回錯。

#### Scenario: Historical VaR on a known series

- **WHEN** 報酬為 `{−0.05, −0.02, 0.00, 0.01, 0.03}`、`confidence: 0.8`、`VaRHistorical`
- **THEN** VaR 等於 `−quantileType7(sorted, 20)` 且與 `insyra.NewDataList(...).Percentile(20)` 取負後一致

#### Scenario: Historical CVaR is at least VaR

- **WHEN** 任意合法報酬序列與 `confidence`
- **THEN** `ConditionalValueAtRisk >= ValueAtRisk`（浮點容差內）

#### Scenario: Parametric VaR on a normal-like series

- **WHEN** `mean = 0.001`、`sd = 0.02` 的手算序列、`confidence: 0.95`
- **THEN** VaR 等於 `−(0.001 + NormPPF(0.05)·0.02)`（1e-12 內）

#### Scenario: Invalid confidence is refused

- **WHEN** `confidence` 為 0、1 或 1.5
- **THEN** 回傳提及 confidence 範圍的錯誤

### Requirement: Downside and benchmark-relative ratios

`quant` SHALL 提供 `SortinoRatio(returns insyra.IDataList, minimumAcceptableReturn, periodsPerYear float64) (float64, error)`：`mean(r − MAR) / downsideDeviation · √periodsPerYear`，downside deviation 為 `sqrt(mean(min(r − MAR, 0)²))`，分母對全部期間取平均。`CalmarRatio(equity insyra.IDataList, days int) (float64, error)`：`AnnualizedReturn(equity, days) / MaxDrawdown(equity)`。`InformationRatio(returns, benchmark insyra.IDataList, periodsPerYear float64) (float64, error)`：`mean(active) / sampleStd(active) · √periodsPerYear`，`active = returns − benchmark`。downside deviation、最大回撤、tracking error 為 0 時 SHALL 回錯；`periodsPerYear <= 0`、長度不同、少於 2 筆、不可讀格子 SHALL 回錯。

#### Scenario: Sortino uses all periods in the denominator

- **WHEN** 報酬 `{0.02, −0.01, 0.03, −0.02}`、`MAR: 0`、`periodsPerYear: 1`
- **THEN** 結果為 `mean / sqrt((0 + 0.0001 + 0 + 0.0004)/4)`（1e-12 內）

#### Scenario: Sortino equals Sharpe when no downside exists is refused

- **WHEN** 所有報酬都高於 MAR
- **THEN** 回傳指出 downside deviation 為零的錯誤

#### Scenario: Calmar composes existing functions

- **WHEN** 任意合法 equity 與 `days`
- **THEN** 結果等於 `AnnualizedReturn(equity, days) / MaxDrawdown(equity)`

#### Scenario: Information ratio of a series against itself is refused

- **WHEN** `returns == benchmark`
- **THEN** 回傳指出 tracking error 為零的錯誤

### Requirement: Drawdown series

`quant` SHALL 提供 `DrawdownSeries(equity insyra.IDataList) (*insyra.DataList, error)`，回傳與輸入等長的非負回撤比例序列 `1 − equity[t] / runningPeak[t]`；running peak 非正的位置輸出 nil。`max` 該序列 SHALL 等於 `MaxDrawdown(equity)`。空序列與不可讀格子 SHALL 回錯。

#### Scenario: Maximum of the series equals MaxDrawdown

- **WHEN** 任意合法 equity
- **THEN** `DrawdownSeries` 的最大值等於 `MaxDrawdown(equity)`（1e-12 內）

#### Scenario: Monotone equity has zero drawdown everywhere

- **WHEN** equity 單調不減
- **THEN** 序列每一格為 0
