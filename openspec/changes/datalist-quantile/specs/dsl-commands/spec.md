## MODIFIED Requirements

### Requirement: DataList statistics commands
系統 SHALL 提供 DataList 統計命令：`sum`、`mean`、`median`、`mode`、`stdev`、`var`、`min`、`max`、`range`、`quartile`、`iqr`、`percentile`、`quantile`、`count`、`counter`。

#### Scenario: Mean of DataList
- **WHEN** 使用者執行 `mean d1`
- **THEN** 系統呼叫 `d1.Mean()` 並顯示結果

#### Scenario: Quartile
- **WHEN** 使用者執行 `quartile d1 1`
- **THEN** 系統呼叫 `d1.Quartile(1)` 顯示第 1 四分位數

#### Scenario: Quantile
- **WHEN** 使用者執行 `quantile d1 0.75`
- **THEN** 系統呼叫 `d1.Quantile(0.75)`，顯示與 `percentile d1 75` 相同的數字

#### Scenario: Counter
- **WHEN** 使用者執行 `counter d1`
- **THEN** 系統呼叫 `d1.Counter()` 顯示頻率表
