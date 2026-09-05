# datalist-numeric-input Specification

## Purpose
Defines how the numeric transforms and reducers on `DataList` treat cells they cannot read: they scan before writing, never substitute `0`, and leave the list untouched on failure. Missing cells (nil / NaN) pass through the in-place transforms and get a NaN rank; the smoothing and interpolation methods require a fully numeric series.

## Requirements
### Requirement: In-place numeric transforms refuse unreadable cells before writing

`DataList.Normalize`、`Standardize`、`ClearOutliers`、`Difference`、`FillNaNWithMean` SHALL 在寫入任何格子之前掃描整份資料。nil 與 NaN 格子 SHALL 視為缺值、保留原樣（`Difference` 對含缺值的相鄰對 SHALL 輸出 NaN）。任一格子既非數值也非缺值時，呼叫 SHALL 設定 `Err()`、SHALL NOT 改動任何格子，且 SHALL 維持該函式既有的失敗回傳形狀。全為數值的輸入 SHALL 得到與變更前逐位元相同的結果。

#### Scenario: Normalize on mixed data leaves the list untouched

- **WHEN** `dl := NewDataList(1, "x", 3); dl.Normalize()`
- **THEN** `dl.Data()` 仍為 `[1, "x", 3]`，`dl.Err()` 非 nil 且訊息含 `"x"` 所在位置

#### Scenario: Standardize keeps NaN cells and standardizes the rest

- **WHEN** `NewDataList(1.0, math.NaN(), 3.0).Standardize()`
- **THEN** 第 2 格仍為 NaN，第 1、3 格為 `(v - mean) / stdev`，`Err()` 為 nil

#### Scenario: ClearOutliers keeps nil cells

- **WHEN** 含 nil 格子與一個明顯離群值的 list 呼叫 `ClearOutliers(2)`
- **THEN** 離群值被移除，nil 格子保留，`Err()` 為 nil

#### Scenario: Fully numeric input is unchanged

- **WHEN** 既有測試中的全數值輸入呼叫五個函式
- **THEN** 結果與變更前相同，既有測試不修改即通過

### Requirement: Rank, smoothing and interpolation never treat a cell as zero

`DataList.Rank` SHALL 對 nil 與 NaN 格子輸出 NaN 名次且不佔用名次位置，對其他非數值格子 SHALL 設定 `Err()` 並回傳 nil。`ExponentialSmoothing`、`DoubleExponentialSmoothing` 與六個 `*Interpolation` 方法 SHALL 要求整份資料皆為有限數值；任一格子為非數值、nil 或 NaN 時 SHALL 設定 `Err()`，平滑方法回傳 nil、插值方法回傳 NaN。任何路徑 SHALL NOT 以 0 代入不可讀的格子。

#### Scenario: Rank refuses a string cell

- **WHEN** `NewDataList(3, "b", 1).Rank()`
- **THEN** 回傳 nil，`Err()` 非 nil

#### Scenario: Rank keeps NaN positions

- **WHEN** `NewDataList(3.0, math.NaN(), 1.0).Rank()`
- **THEN** 資料為 `[2, NaN, 1]`

#### Scenario: Interpolation refuses a nil cell

- **WHEN** `NewDataList(1.0, nil, 3.0).LinearInterpolation(0.5)`
- **THEN** 回傳 NaN，`Err()` 非 nil，訊息指出第 2 格

#### Scenario: Exponential smoothing refuses a string cell

- **WHEN** `NewDataList(1, "2", 3).ExponentialSmoothing(0.5)`
- **THEN** 回傳 nil，`Err()` 非 nil

