# quant-portfolio Specification

## Purpose
`quant` 的均值變異數投組最適化：在權重和為 1 與逐資產上下界（預設 long-only）下求最小變異數、目標報酬或最大 Sharpe，另有效率前緣掃描與自備動差的入口；純 Go 加速投影梯度求解，正確性以閉式解、格點窮舉與 opt-in 的 cvxpy 對照釘住，未收斂以旗標回報。

## Requirements
### Requirement: Mean-variance optimisation under sum-to-one and box constraints

`quant` SHALL 提供 `PortfolioObjective`（`MinimumVariance`、`TargetReturn`、`MaximumSharpe`）、`PortfolioConfig{Objective; TargetReturn, RiskFreeRate float64; MinWeight, MaxWeight []float64; Tolerance float64; MaxIterations int}`、`PortfolioResult{Weights []float64; AssetNames []string; ExpectedReturn, Variance, Volatility, SharpeRatio float64; Iterations int; Converged bool}` 與方法 `Weight(name string) (float64, bool)`，以及 `OptimizePortfolio(returns insyra.IDataTable, cfg PortfolioConfig) (*PortfolioResult, error)` 與 `OptimizePortfolioMoments(mean []float64, cov [][]float64, names []string, cfg PortfolioConfig) (*PortfolioResult, error)`。權重 SHALL 滿足 `Σw = 1` 與 `MinWeight[i] ≤ w[i] ≤ MaxWeight[i]`（未給時為 `[0, 1]`）。`OptimizePortfolio` SHALL 以各欄平均為期望報酬、樣本共變異數（n−1）為 Σ。`MinimumVariance` SHALL 最小化 `wᵀΣw`；`TargetReturn` SHALL 在 `μᵀw = TargetReturn` 下最小化 `wᵀΣw`；`MaximumSharpe` SHALL 最大化 `(μᵀw − RiskFreeRate) / sqrt(wᵀΣw)`。`SharpeRatio` 欄位 SHALL 為每期值。`Tolerance` 預設 1e-10、`MaxIterations` 預設 10000；未收斂 SHALL 回傳 `Converged: false` 與當時最佳解，不回錯。

#### Scenario: Interior minimum variance matches the closed form

- **WHEN** 三資產、Σ 正定、閉式解 `Σ⁻¹1 / 1ᵀΣ⁻¹1` 每個分量都在 (0, 1) 內，以 `MinimumVariance` 呼叫
- **THEN** `Weights` 與閉式解逐一相差不超過 1e-8，`Converged == true`

#### Scenario: Interior target return matches the closed form

- **WHEN** 同上資料、目標報酬使閉式解在內部，以 `TargetReturn` 呼叫
- **THEN** `Weights` 與拉格朗日閉式解相差不超過 1e-8，`ExpectedReturn` 等於目標（1e-10）

#### Scenario: Long-only case matches exhaustive grid search

- **WHEN** 三資產、閉式解含負權重，以預設 `[0, 1]` 界呼叫 `MinimumVariance`
- **THEN** `Variance` 不大於步長 1e-3 的 simplex 格點窮舉最小值，且 `Weights` 全在 `[0, 1]`、和為 1（1e-12）

#### Scenario: Box bounds are respected

- **WHEN** `MaxWeight = [0.5, 0.5, 0.5]`、`MinWeight = [0.1, 0, 0]`
- **THEN** 每個權重在界內（1e-12），和為 1

#### Scenario: Maximum Sharpe is no worse than any frontier point

- **WHEN** 以 `MaximumSharpe` 求解，並以 `EfficientFrontier` 取 50 點
- **THEN** 結果的 `SharpeRatio` 不小於 50 點中的最大值減 1e-6

#### Scenario: Moments entry point agrees with the table entry point

- **WHEN** 以同一組報酬先算均值與共變異數再呼叫 `OptimizePortfolioMoments`
- **THEN** 權重與 `OptimizePortfolio` 相同（1e-12）

#### Scenario: Weight lookup by name

- **WHEN** 欄名為 `A, B, C`
- **THEN** `Weight("B")` 回傳第二個權重與 `true`；`Weight("Z")` 回傳 `0, false`

### Requirement: Efficient frontier sweep

`quant` SHALL 提供 `EfficientFrontier(returns insyra.IDataTable, points int, cfg PortfolioConfig) ([]PortfolioResult, error)`：在最小變異數組合的報酬與界內可達最大報酬之間等距取 `points` 個目標報酬，各以 `TargetReturn` 求解，依 `ExpectedReturn` 遞增回傳。`points < 2` SHALL 回錯。

#### Scenario: Frontier is monotone

- **WHEN** 取 20 點
- **THEN** `ExpectedReturn` 嚴格遞增，`Variance` 非遞減

#### Scenario: Endpoints

- **WHEN** 取 20 點
- **THEN** 第一點的 `Variance` 等於 `MinimumVariance` 解的 `Variance`（1e-8），最後一點的 `ExpectedReturn` 等於界內可達最大報酬（1e-8）

### Requirement: Input validation and refusal

`OptimizePortfolio` SHALL 在下列情況回錯：nil 或少於 2 欄；觀察數少於欄數加 1；任一格子不可讀、NaN、Inf（錯誤含欄名與列號）；`MinWeight`／`MaxWeight` 長度不符或 `lo > hi`；`Σlo > 1` 或 `Σhi < 1`（不可行）；`TargetReturn` 超出界內可達範圍；未知 `Objective`。`OptimizePortfolioMoments` SHALL 另外拒絕維度不一致、非對稱或非半正定的 `cov`。

#### Scenario: Infeasible bounds

- **WHEN** `MaxWeight = [0.3, 0.3, 0.3]`
- **THEN** 回傳提及 `Σhi < 1` 或 infeasible 的錯誤

#### Scenario: Unattainable target return

- **WHEN** `TargetReturn` 高於界內最大可達報酬
- **THEN** 回傳指出可達範圍的錯誤

#### Scenario: Unreadable cell names the column

- **WHEN** 欄 `B` 第 4 列為 `"n/a"`
- **THEN** 錯誤含 `B` 與 `row 4`

#### Scenario: Non-PSD covariance is refused

- **WHEN** `cov = [[1, 2], [2, 1]]`
- **THEN** `OptimizePortfolioMoments` 回傳指出 covariance 非半正定的錯誤

### Requirement: Cross-language agreement is available on demand

在 `INSYRA_RUN_CVXPY=1` 時，測試 SHALL 以 crosslang venv 的 Python 執行 `cvxpy`，對隨機的 long-only 與 box-bounded 問題比較目標值；未設定時 SHALL Skip。目標值差異 SHALL 在 1e-6 內。

#### Scenario: Gated by default

- **WHEN** 未設定環境變數
- **THEN** 該測試 Skip，`go test ./quant/...` 不需要 Python

