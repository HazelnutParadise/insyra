# cli-quant-portfolio Specification

## Purpose
CLI 的 `quant portfolio` 與 `quant frontier` 形式：以資產報酬 DataTable 呼叫 `OptimizePortfolio`／`EfficientFrontier`，目標關鍵字、`rf`、逐資產 `min`／`max` 界在呼叫前驗證，權重存成 Asset／Weight 表加 `_stats`，前緣存成每點一列的寬表。

## Requirements
### Requirement: quant portfolio form

`quant` SHALL 接受 `quant portfolio <returns_dt> minvar|target <r>|maxsharpe [rf <r>] [min <v1,v2,…>] [max <v1,v2,…>] [as <var>]`，呼叫 `quant.OptimizePortfolio`。`minvar`→`MinimumVariance`、`target <r>`→`TargetReturn` 並帶 `TargetReturn`、`maxsharpe`→`MaximumSharpe`；`rf` 對應 `RiskFreeRate`（預設 0）；`min`／`max` 為逗號分隔的逐資產界（欄序），對應 `MinWeight`／`MaxWeight`，未給時交由函式庫預設。輸出 SHALL 每資產印一行 `<asset>=<weight>` 與一行摘要（`return= vol= sharpe= iterations= converged=`）。SHALL 存 `DataTable`（欄 `Asset`、`Weight`，每資產一列）到 `as` 或 `$result`，並另存一列 `<var>_stats`（欄 `ExpectedReturn`、`Variance`、`Volatility`、`SharpeRatio`、`Iterations`、`Converged`）。界的長度與欄數不符、界含非數值、`target` 缺數值、未知目標關鍵字 SHALL 在呼叫前回傳含 `quant portfolio:` 前綴的錯誤；函式庫錯誤 SHALL 加同一前綴原樣回傳。未收斂 SHALL 不回錯。

#### Scenario: Minimum variance matches the library

- **WHEN** `dt` 為三資產報酬表，執行 `quant portfolio dt minvar as w`
- **THEN** `w` 的 `Weight` 欄逐一等於 `quant.OptimizePortfolio(dt, PortfolioConfig{}).Weights`（1e-12），`Asset` 欄等於表格欄名，`w_stats` 的 `Variance` 等於結果的 `Variance`

#### Scenario: Target return and bounds are passed through

- **WHEN** 執行 `quant portfolio dt target 0.001 rf 0.0001 min 0.1,0,0 max 0.5,0.5,0.5 as w`
- **THEN** 結果等於 `OptimizePortfolio(dt, PortfolioConfig{Objective: TargetReturn, TargetReturn: 0.001, RiskFreeRate: 0.0001, MinWeight: [0.1,0,0], MaxWeight: [0.5,0.5,0.5]})`（1e-12）

#### Scenario: Maximum Sharpe

- **WHEN** 執行 `quant portfolio dt maxsharpe rf 0.0001 as w`
- **THEN** 結果等於 `OptimizePortfolio(dt, PortfolioConfig{Objective: MaximumSharpe, RiskFreeRate: 0.0001})`（1e-12）

#### Scenario: Bounds length mismatch is refused before the call

- **WHEN** 三資產表但 `min 0.1,0`
- **THEN** 回傳含 `quant portfolio:` 與欄數的錯誤

#### Scenario: Library error is surfaced

- **WHEN** `max 0.3,0.3,0.3`（不可行）
- **THEN** 錯誤以 `quant portfolio:` 開頭並含函式庫訊息

### Requirement: quant frontier form

`quant` SHALL 接受 `quant frontier <returns_dt> <points> [rf <r>] [min …] [max …] [as <var>]`，呼叫 `quant.EfficientFrontier`。SHALL 存 `DataTable`，每點一列，欄依序為 `ExpectedReturn`、`Variance`、`Volatility`、`SharpeRatio`、`Converged`，之後每資產一欄（以資產名為欄名）放權重。`points` 非整數或小於 2 SHALL 回錯。

#### Scenario: Frontier table matches the library

- **WHEN** 執行 `quant frontier dt 10 as f`
- **THEN** `f` 有 10 列，`ExpectedReturn` 欄與 `EfficientFrontier(dt, 10, PortfolioConfig{})` 的各點相等（1e-12），資產權重欄名等於表格欄名

#### Scenario: Invalid points

- **WHEN** 執行 `quant frontier dt 1`
- **THEN** 回傳含 `quant frontier:` 前綴的錯誤

