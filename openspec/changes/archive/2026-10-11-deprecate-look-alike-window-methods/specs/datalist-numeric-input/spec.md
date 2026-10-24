## ADDED Requirements

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
