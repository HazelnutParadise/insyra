# stats-moment-input Specification

## Purpose
States the input contract for `stats.Skewness` and `stats.Kurtosis`: a non-numeric or non-finite cell is refused with an error naming the row, the same rule every other `stats` entry point follows.

## Requirements
### Requirement: Skewness and Kurtosis refuse unreadable input

`stats.Skewness` 與 `stats.Kurtosis` SHALL 經由套件的 `numericValues` 讀取輸入。任一格子非數值、NaN 或 Inf 時 SHALL 回傳錯誤，訊息 SHALL 含標籤 `sample` 與從 1 起算的列號，且 SHALL NOT 以 0 代入。全為有限數值的輸入 SHALL 得到與變更前逐位元相同的結果；既有錯誤（空資料、方法數過多、n 不足、零變異）SHALL 維持原訊息。

#### Scenario: A blank cell is refused

- **WHEN** `Skewness(insyra.NewDataList(1.0, nil, 3.0, 4.0))`
- **THEN** 回傳含 `sample` 與 `row 2` 的錯誤

#### Scenario: A string cell is refused by Kurtosis

- **WHEN** `Kurtosis([]any{1.0, "x", 3.0, 4.0})`
- **THEN** 回傳含 `sample` 與 `row 2` 的錯誤

#### Scenario: R reference values are unchanged

- **WHEN** 既有 `TestSkewness_R` 與 `TestKurtosis_R` 執行
- **THEN** 不修改即通過

