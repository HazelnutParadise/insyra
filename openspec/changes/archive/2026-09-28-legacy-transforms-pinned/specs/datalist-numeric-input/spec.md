## ADDED Requirements

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
