# stats-test-input Specification

## Purpose
The parametric tests (t, z, F, Bartlett, Levene) and `CalculateMoment` read every observation through `numericSlice`, so an unreadable cell is an error naming its row and is never counted in n.

## Requirements
### Requirement: Parametric tests refuse unreadable cells

`SingleSampleTTest`、`TwoSampleTTest`、`SingleSampleZTest`、`TwoSampleZTest`、`FTestForVarianceEquality`、`BartlettTest`、`LeveneTest` 與 `CalculateMoment` SHALL 經 `numericSlice` 讀取輸入；任一格非數值、nil、NaN 或 Inf 時 SHALL 回傳含標籤與列號的錯誤，SHALL NOT 把該格算進 n。全為有限數值的輸入 SHALL 得到與變更前逐位元相同的統計量。

#### Scenario: Blank cell is refused
- **WHEN** `SingleSampleTTest(NewDataList(1.0, 2.0, nil, 3.0), 0)`
- **THEN** 回傳錯誤，訊息含 `row 3`

#### Scenario: Clean input unchanged
- **WHEN** 既有測試的全數值輸入
- **THEN** 既有測試不修改即通過

