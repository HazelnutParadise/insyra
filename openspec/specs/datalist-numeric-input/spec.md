# datalist-numeric-input Specification

## Purpose
同一個值在「是不是數字」的判斷和「轉成 float64」之間得到一致的答案，以數值型別為底的自訂型別也一樣。

## Requirements

### Requirement: A named numeric type is a number everywhere or nowhere

判斷「這是不是數字」與「把它轉成 float64」的兩條路徑 SHALL 對同一個值給出一致的答案。以數值 kind 為底的具名型別（`type Celsius float64`）SHALL 兩邊都接受。

#### Scenario: A user-defined numeric type
- **WHEN** 對 `Celsius(36.6)` 呼叫 `IsNumeric` 與 `ToFloat64Safe`
- **THEN** 兩者都說是數字，且轉換結果為 36.6

### Requirement: Four look-alike window methods are deprecated and keep their meaning

`DataList.Difference`, `MovingAverage`, `MovingStdev` and `WeightedMovingAverage` SHALL be marked Deprecated in their doc comments and on `IDataList`, naming their replacements (`Diff(1)`, `Rolling(RollingOptions{Window: w}).Mean()`, `Rolling(RollingOptions{Window: w}).Std()`, `Rolling(RollingOptions{Window: w, Weights: ws}).Mean()`), how the replacement's result lines up with its own, and that they are removed in 0.4.

#### Scenario: The deprecation is written where a caller sees it
- **WHEN** 讀取 `Difference`、`MovingAverage`、`MovingStdev`、`WeightedMovingAverage` 的 doc comment
- **THEN** 每一個都有以 `Deprecated: use` 開頭、指名替代呼叫的段落，並寫明在 0.4 移除

#### Scenario: The old meaning stays
- **WHEN** 對 `[1, 2, 3, 4, 5]` 呼叫 `MovingAverage(3)`
- **THEN** 結果仍為 `[2, 3, 4]`，而不是 `Rolling` 的 `[nil, nil, 2, 3, 4]`

#### Scenario: ExponentialSmoothing is not deprecated
- **WHEN** 讀取 `ExponentialSmoothing` 的 doc comment
- **THEN** 沒有 Deprecated 段落

#### Scenario: CLI commands keep output
- **WHEN** 執行 CLI 的 `movavg` 與 `diff`
- **THEN** 輸出維持不變

### Requirement: The older window and fill methods keep contracts of their own

`DataList.Difference`, `FillNaNWithMean`, `MovingAverage`, `MovingStdev`, `ExponentialSmoothing` and `WeightedMovingAverage` SHALL keep their own contracts distinct from newer replacements, matching replacement values on fully numeric series where positions overlap while preserving historical differences.

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

#### Scenario: Differences pinned by regression tests
- **WHEN** 執行測試套件驗證舊版方法與新版替代方案的所有文件差異
- **THEN** 任何未經文件重新檢討的行為變更都會導致測試失敗
