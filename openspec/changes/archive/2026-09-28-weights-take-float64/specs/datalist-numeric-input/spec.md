## ADDED Requirements

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
