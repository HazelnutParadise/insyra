## MODIFIED Requirements

### Requirement: In-place numeric transforms refuse unreadable cells before writing

`DataList.Normalize`、`Standardize`、`ClearOutliers` SHALL 在寫入任何格子之前掃描整份資料。nil 與 NaN 格子 SHALL 視為缺值、保留原樣。任一格子既非數值也非缺值時，呼叫 SHALL 設定 `Err()`、SHALL NOT 改動任何格子，且 SHALL 維持該函式既有的失敗回傳形狀。全為數值的輸入 SHALL 得到與變更前逐位元相同的結果。

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

- **WHEN** 既有測試中的全數值輸入呼叫三個函式
- **THEN** 結果與變更前相同，既有測試不修改即通過

### Requirement: The weighted methods keep the rest of their behaviour

A weights slice whose length does not match the list's length SHALL make `WeightedMean` record an error and return `NaN`. `WeightedMean` SHALL skip a non-numeric element together with its weight and SHALL return `NaN` for an empty list, for a list with no numeric element and for a zero total weight.

#### Scenario: Weights of the wrong length
- **WHEN** `NewDataList(1.0, 2.0, 3.0).WeightedMean([]float64{1, 2})`
- **THEN** 回傳 NaN，並在接收者記錄錯誤

#### Scenario: A non-numeric element is skipped with its weight
- **WHEN** `NewDataList(1, "x", 3).WeightedMean([]float64{1, 100, 1})`
- **THEN** 結果為 2，權重 100 不計入

## REMOVED Requirements

### Requirement: Four look-alike window methods are deprecated and keep their meaning
**Reason**: `DataList.Difference`, `MovingAverage`, `MovingStdev` and `WeightedMovingAverage` were removed from `DataList` and `IDataList` in 0.4, the release after the one that deprecated them.
**Migration**: Use `Diff(1)`, `Rolling(RollingOptions{Window: w}).Mean()`, `Rolling(RollingOptions{Window: w}).Std()` and `Rolling(RollingOptions{Window: w, Weights: ws}).Mean()`; the results line up with the list's own length and carry nil where the old methods returned a shorter list.

### Requirement: The deprecated window methods behave as before until removed
**Reason**: The methods it kept alive were removed in 0.4. Its two other rules, that `ExponentialSmoothing` is not deprecated and that the CLI's `movavg` and `diff` keep their output, move to the added requirement below.
**Migration**: None.

### Requirement: A NaN-only mean fill has a replacement that keeps its behaviour
**Reason**: `DataList.FillNaNWithMean` was removed in 0.4, so its Deprecated notice no longer exists.
**Migration**: Use `dl.ReplaceNaNsWith(dl.Clone().ClearNilsAndNaNs().Mean())` to fill `NaN` alone, or `FillWithMean`, which also fills `nil`.

### Requirement: Weights are a float64 slice
**Reason**: It named deprecated APIs that were removed in 0.4, and its scenarios cannot be edited in place; it is restated under a new name in this change.
**Migration**: See the requirement "WeightedMean takes float64 weights".

### Requirement: The older window and fill methods keep contracts of their own
**Reason**: It named deprecated APIs that were removed in 0.4, and its scenarios cannot be edited in place; it is restated under a new name in this change.
**Migration**: See the requirement "ExponentialSmoothing keeps a contract of its own".

## ADDED Requirements

### Requirement: ExponentialSmoothing stays and the CLI time-series commands keep their output

`DataList.ExponentialSmoothing` SHALL NOT be deprecated. The CLI's `movavg` and `diff` SHALL keep their output: `movavg` the shorter list of window means, `diff` the `n - 1` differences, each computed by the command itself.

#### Scenario: ExponentialSmoothing is not deprecated
- **WHEN** 讀取 `ExponentialSmoothing` 的 doc comment
- **THEN** 沒有 Deprecated 段落

#### Scenario: The commands keep their shorter output
- **WHEN** 對 `[1, 2, 3, 4, 5]` 執行 `movavg d1 3` 與 `diff d1`
- **THEN** 結果分別為 `[2, 3, 4]` 與 `[1, 1, 1, 1]`

### Requirement: WeightedMean takes float64 weights

`DataList.WeightedMean` SHALL take its weights as `[]float64`, the same type as `RollingOptions.Weights`. `IDataList` SHALL declare the same signature.

#### Scenario: A float64 slice of weights
- **WHEN** `NewDataList(1, 2, 3, 4).WeightedMean([]float64{1, 2, 3, 4})`
- **THEN** 結果為 3，`Err()` 為 nil

#### Scenario: The signature is checked by the compiler
- **WHEN** 程式把 `*DataList` 或 `[]int` 當權重傳入
- **THEN** 編譯失敗，而不是在執行時才以長度不符報錯

### Requirement: ExponentialSmoothing keeps a contract of its own

`DataList.ExponentialSmoothing` SHALL keep its own contract, which differs from that of `EWM(EWMOptions{Alpha: a}).Mean()`. On a fully numeric series it SHALL give its replacement's numbers wherever both give one. A test SHALL pin each documented difference.

#### Scenario: ExponentialSmoothing accepts alpha zero and refuses a gap
- **WHEN** 對 `[1, 2, 3]` 呼叫 `ExponentialSmoothing(0)` 與 `EWM(EWMOptions{Alpha: 0}).Mean()`
- **THEN** 前者為 `[1, 1, 1]`，後者回傳長度 0 並記錄錯誤
- **AND** 對 `[1, nil, 3]` 以 `alpha` 0.5，前者記錄錯誤，後者為 `[1, 1, 2.333…]`
