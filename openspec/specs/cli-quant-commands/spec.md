# cli-quant-commands Specification

## Purpose
CLI／REPL／DSL 的 `quant` 命令群：每個 `quant` 函式一個形式（績效比率、尾端風險、市場曝險、多因子歸因、選擇權定價），引數一對一對應 Go 參數，純量印出並存變數，結構化結果存成 DataTable，函式庫錯誤加前綴原樣回傳。

## Requirements
### Requirement: quant command forms map one-to-one to quant functions

CLI SHALL 提供 `quant <form> ...` 命令，各形式 SHALL 呼叫對應的 `quant` 函式並回傳其結果：`sharpe`→`SharpeRatio`、`sortino`→`SortinoRatio`、`ir`→`InformationRatio`、`maxdd`→`MaxDrawdown`、`annret`→`AnnualizedReturn`、`calmar`→`CalmarRatio`、`drawdown`→`DrawdownSeries`、`var`→`ValueAtRisk`、`cvar`→`ConditionalValueAtRisk`、`beta`→`Beta`、`capm`→`CAPM`、`factor`→`FactorModel`、`bs`→`BlackScholes`、`iv`→`ImpliedVolatility`。序列引數 SHALL 為 `DataList` 變數（`factor` 的 `factors` 為 `DataTable`）；`periods`、`days`、`confidence` SHALL 為必填位置引數；`rf`、`mar`、`q` 為 `key value` 選項，預設 0；VaR 方法預設 `historical`。函式庫錯誤 SHALL 原樣回傳並加 `quant <form>:` 前綴。未知形式 SHALL 回傳列出所有形式的錯誤。

#### Scenario: Sharpe equals the library

- **WHEN** 變數 `r` 為報酬序列，執行 `quant sharpe r 252 rf 0.0001 as s`
- **THEN** 輸出含 `sharpe=`，`s` 等於 `quant.SharpeRatio(r, 0.0001, 252)`

#### Scenario: VaR method keyword

- **WHEN** 執行 `quant var r 0.95 parametric as v`
- **THEN** `v` 等於 `quant.ValueAtRisk(r, 0.95, VaRParametric)`；省略關鍵字時等於歷史法

#### Scenario: Missing required positional

- **WHEN** 執行 `quant sharpe r`
- **THEN** 回傳含該形式用法的錯誤

#### Scenario: Library error is surfaced

- **WHEN** `r` 含一個 nil，執行 `quant sortino r 252`
- **THEN** 錯誤訊息以 `quant sortino:` 開頭並含函式庫的列號訊息

### Requirement: Structured results are printed and stored as tables

`quant capm` SHALL 印出 `beta`、`alpha`、`r2`、`beta_se`、`alpha_se`、`n`，並在 `as` 時存一列 `DataTable`（同名欄）。`quant factor` SHALL 印出 `alpha` 與每個因子的 `exposure`、`t`、`p`，並在 `as` 時存 `DataTable`（欄 `Factor, Exposure, StdErr, TValue, PValue`，一因子一列），`alpha` 及其推論值 SHALL 以 `alpha` 名稱另存為 `<var>_alpha` 一列表。`quant bs` SHALL 印出 `price delta gamma vega theta rho` 並在 `as` 時存一列 `DataTable`。`quant drawdown` SHALL 存 `DataList`。

#### Scenario: CAPM table

- **WHEN** 執行 `quant capm a m rf 0.0002 as c`
- **THEN** `c` 為一列 `DataTable`，`beta` 欄等於 `quant.CAPM(a, m, 0.0002).Beta`

#### Scenario: Factor table rows follow factor order

- **WHEN** `f` 有 `MKT, SMB, HML` 三欄，執行 `quant factor a f as fm`
- **THEN** `fm` 有三列，`Factor` 欄依序為 `MKT, SMB, HML`，`Exposure` 等於 `FactorModel` 的 `Exposures`；`fm_alpha` 存在

#### Scenario: Black–Scholes from scalars

- **WHEN** 執行 `quant bs call 42 40 0.10 0.20 0.5 as o`
- **THEN** 輸出 `price=4.759…`，`o` 的 `price` 欄等於 `BlackScholes` 的 `Price`

#### Scenario: Implied volatility round trip

- **WHEN** 以 `quant bs` 的價格執行 `quant iv call <price> 42 40 0.10 0.5 as v`
- **THEN** `v` 等於 0.20（1e-8 內）

