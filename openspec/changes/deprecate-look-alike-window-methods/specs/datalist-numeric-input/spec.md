## ADDED Requirements

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
