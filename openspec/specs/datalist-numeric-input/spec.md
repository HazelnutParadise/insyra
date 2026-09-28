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

### Requirement: A named numeric type is a number everywhere or nowhere

判斷「這是不是數字」與「把它轉成 float64」的兩條路徑 SHALL 對同一個值給出一致的答案。以數值 kind 為底的具名型別（`type Celsius float64`）SHALL 兩邊都接受。

#### Scenario: A user-defined numeric type
- **WHEN** 對 `Celsius(36.6)` 呼叫 `IsNumeric` 與 `ToFloat64Safe`
- **THEN** 兩者都說是數字，且轉換結果為 36.6

### Requirement: Weights are a float64 slice

`DataList.WeightedMean` SHALL take its weights as `[]float64`, and `DataList.WeightedMovingAverage` SHALL take `(windowSize int, weights []float64)`, the same type as `RollingOptions.Weights`. `IDataList` SHALL declare the same signatures. The other behaviour of both methods SHALL stay as it was: a weights slice whose length does not match (the list's length for `WeightedMean`, `windowSize` for `WeightedMovingAverage`) SHALL record an error, `WeightedMean` then returning `NaN` and `WeightedMovingAverage` an empty list carrying the error; `WeightedMean` SHALL skip a non-numeric element together with its weight and SHALL return `NaN` for an empty list, for a list with no numeric element and for a zero total weight.

#### Scenario: A float64 slice of weights
- **WHEN** `NewDataList(1, 2, 3, 4).WeightedMean([]float64{1, 2, 3, 4})`
- **THEN** 結果為 3，`Err()` 為 nil

#### Scenario: Weighted moving average with float64 weights
- **WHEN** `NewDataList(1.0, 2.0, 3.0).WeightedMovingAverage(2, []float64{1, 3})`
- **THEN** 結果為 `[1.75, 2.75]`

#### Scenario: Weights of the wrong length
- **WHEN** `NewDataList(1.0, 2.0, 3.0).WeightedMean([]float64{1, 2})`，以及 `WeightedMovingAverage(2, []float64{1})`
- **THEN** 前者回傳 NaN、後者回傳長度 0 的 list，兩者都在接收者記錄錯誤

#### Scenario: A non-numeric element is skipped with its weight
- **WHEN** `NewDataList(1, "x", 3).WeightedMean([]float64{1, 100, 1})`
- **THEN** 結果為 2，權重 100 不計入

#### Scenario: The signature is checked by the compiler
- **WHEN** 程式把 `*DataList` 或 `[]int` 當權重傳入
- **THEN** 編譯失敗，而不是在執行時才以長度不符報錯

### Requirement: The older window and fill methods keep contracts of their own

`DataList.Difference`, `FillNaNWithMean`, `MovingAverage`, `MovingStdev`, `ExponentialSmoothing` and `WeightedMovingAverage` SHALL keep their own contracts, which differ from those of `Diff(1)`, `FillWithMean`, `Rolling(...).Mean()`, `Rolling(...).Std()`, `EWM(EWMOptions{Alpha: a}).Mean()` and `Rolling(RollingOptions{Weights: ...}).Mean()`. On a fully numeric series the old methods SHALL produce the same numbers as their replacements over the positions both produce. A test SHALL pin every difference the documentation lists, so neither side can change without the documentation being revisited.

#### Scenario: Difference is shorter and strict
- **WHEN** 對 `[1, 4, 9, 16]` 呼叫 `Difference()` 與 `Diff(1)`
- **THEN** 前者為 `[3, 5, 7]`，後者為 `[nil, 3, 5, 7]`
- **AND** 對 `[1, nil, 4]`，前者為 `[NaN, NaN]`，後者為 `[nil, nil, nil]`
- **AND** 對 `["a", 1, 2]`，前者回傳長度 0 並記錄錯誤，後者為 `[nil, nil, 1]` 且無錯誤

#### Scenario: FillNaNWithMean fills NaN only
- **WHEN** 對 `[1, nil, NaN, 3]` 呼叫 `FillNaNWithMean()` 與 `FillWithMean()`
- **THEN** 前者為 `[1.0, nil, 2.0, 3.0]`（全部數字改寫成 `float64`），後者為 `[1, 2.0, 2.0, 3]`

#### Scenario: MovingAverage is shorter and fails on a gap
- **WHEN** 對 `[1, 2, 3, 4, 5]` 以視窗 3 呼叫兩者
- **THEN** `MovingAverage` 為 `[2, 3, 4]`，`Rolling` 為 `[nil, nil, 2, 3, 4]`
- **AND** 對 `[1, nil, 3, 4, 5]` 以視窗 2，`MovingAverage` 回傳長度 0 並記錄錯誤，`Rolling` 為 `[nil, nil, nil, 3.5, 4.5]`
- **AND** 視窗大於長度時 `MovingAverage` 記錄錯誤，`Rolling` 回傳整列 nil

#### Scenario: MovingStdev skips a gap and gives NaN for a window of one
- **WHEN** 對 `[1, 2, 3]` 以視窗 1 呼叫 `MovingStdev` 與 `Rolling(...).Std()`
- **THEN** 前者為三個 `NaN`，後者為三個 nil

#### Scenario: ExponentialSmoothing accepts alpha zero and refuses a gap
- **WHEN** 對 `[1, 2, 3]` 呼叫 `ExponentialSmoothing(0)` 與 `EWM(EWMOptions{Alpha: 0}).Mean()`
- **THEN** 前者為 `[1, 1, 1]`，後者回傳長度 0 並記錄錯誤
- **AND** 對 `[1, nil, 3]` 以 `alpha` 0.5，前者記錄錯誤，後者為 `[1, 1, 2.333…]`

#### Scenario: WeightedMovingAverage divides by a zero weight sum
- **WHEN** 對 `[1, 2, 3]` 以視窗 2、權重 `[1, -1]` 呼叫兩者
- **THEN** `WeightedMovingAverage` 為 `[-Inf, -Inf]`，`Rolling` 為 `[nil, nil, nil]`

### Requirement: Four look-alike window methods are deprecated and keep their meaning

`DataList.Difference`, `MovingAverage`, `MovingStdev` and `WeightedMovingAverage` SHALL be marked Deprecated in their doc comments and on `IDataList`, each naming its replacement (`Diff(1)`, `Rolling(RollingOptions{Window: w}).Mean()`, `Rolling(RollingOptions{Window: w}).Std()`, `Rolling(RollingOptions{Window: w, Weights: ws}).Mean()`), how the replacement's result lines up with its own, and that it is removed in the release after the one that deprecated it. Until then each SHALL keep exactly the behaviour the requirement "The older window and fill methods keep contracts of their own" pins. `ExponentialSmoothing` SHALL NOT be deprecated. The CLI's `movavg` and `diff` SHALL keep their output.

#### Scenario: The deprecation is written where a caller sees it
- **WHEN** 讀取 `Difference`、`MovingAverage`、`MovingStdev`、`WeightedMovingAverage` 的 doc comment
- **THEN** 每一個都有以 `Deprecated: use` 開頭、指名替代呼叫的段落，並寫明下一版移除

#### Scenario: The old meaning stays
- **WHEN** 對 `[1, 2, 3, 4, 5]` 呼叫 `MovingAverage(3)`
- **THEN** 結果仍為 `[2, 3, 4]`，而不是 `Rolling` 的 `[nil, nil, 2, 3, 4]`

#### Scenario: ExponentialSmoothing is not deprecated
- **WHEN** 讀取 `ExponentialSmoothing` 的 doc comment
- **THEN** 沒有 Deprecated 段落

### Requirement: A NaN-only mean fill has a replacement that keeps its behaviour

`FillNaNWithMean`'s Deprecated notice SHALL name `dl.ReplaceNaNsWith(dl.Clone().ClearNilsAndNaNs().Mean())` as the replacement that fills `NaN` alone, instead of only `FillWithMean`, which also fills `nil`. On a list of numbers, `nil` and `NaN`, the replacement SHALL give the same values in every cell as `FillNaNWithMean`, leaving `nil` cells as they are.

#### Scenario: The replacement matches
- **WHEN** 對 `[1, nil, NaN, 3, NaN]` 分別呼叫 `FillNaNWithMean()` 與 `ReplaceNaNsWith(Clone().ClearNilsAndNaNs().Mean())`
- **THEN** 兩者每一格的數值相同：`[1, nil, 2, 3, 2]`，nil 保留
