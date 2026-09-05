# quant-performance-input Specification

## Purpose
`quant` 舊有入口（`SharpeRatio`、`MaxDrawdown`、`AnnualizedReturn`、`DeflatedSharpeRatio`、`PBO`）的輸入契約：一律經 `numericSeries` 讀取，不可讀、NaN、Inf 或 nil 的輸入回傳指出序列與列號的錯誤，不以 0 代入；全數值輸入結果不變。

## Requirements
### Requirement: Legacy quant entry points refuse unreadable input

`SharpeRatio`、`MaxDrawdown`、`AnnualizedReturn`、`DeflatedSharpeRatio`（其 `trialSharpes`）與 `PBO`（每一欄）SHALL 經由 `numericSeries` 讀取輸入。任一格子非數值、`NaN` 或 `Inf` 時 SHALL 回傳錯誤，錯誤訊息 SHALL 含序列標籤（`returns`、`equity`、`trialSharpes`、`PBO` 為 `column <j>`，`j` 從 0 起算與既有訊息一致）與從 1 起算的列號，且 SHALL NOT 以 0 代入。nil 輸入 SHALL 回傳指出序列標籤為 nil 的錯誤。全為有限數值的輸入 SHALL 得到與變更前逐位元相同的結果；既有的驗證錯誤（少於 2 筆、`periodsPerYear` 非正、零波動、空 equity、`nSplits` 規則等）SHALL 維持原訊息。

#### Scenario: Blank in a return series is refused, not zeroed

- **WHEN** `SharpeRatio(insyra.NewDataList(0.01, nil, 0.02), 0, 252)`
- **THEN** 回傳錯誤，訊息含 `returns` 與 `row 2`，不回傳數值

#### Scenario: Text in an equity curve is refused

- **WHEN** `MaxDrawdown(insyra.NewDataList(100.0, "n/a", 90.0))` 與 `AnnualizedReturn(同序列, 30)`
- **THEN** 兩者都回傳含 `equity` 與 `row 2` 的錯誤

#### Scenario: NaN trial Sharpe is refused

- **WHEN** `DeflatedSharpeRatio(1.0, 100, 0, 3, insyra.NewDataList(0.5, math.NaN(), 0.7))`
- **THEN** 回傳含 `trialSharpes` 與 `row 2` 的錯誤

#### Scenario: PBO names the column and row

- **WHEN** `PBO` 的第 2 欄（`j == 1`）第 3 列為 `"x"`
- **THEN** 回傳含 `column 1` 與 `row 3` 的錯誤

#### Scenario: Numeric input is unchanged

- **WHEN** 以既有測試中的全數值輸入呼叫五個函式
- **THEN** 結果與變更前相同，既有測試不修改即通過

#### Scenario: Nil input

- **WHEN** `SharpeRatio(nil, 0, 252)`
- **THEN** 回傳指出 `returns is nil` 的錯誤而不是 panic

