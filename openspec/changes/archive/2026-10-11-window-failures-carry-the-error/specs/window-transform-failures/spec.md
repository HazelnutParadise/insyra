## ADDED Requirements

### Requirement: DataList window reducers hand failures to empty results

When `DataList.Rolling` or `DataList.EWM` cannot build its view, every reducer of the returned view SHALL return an empty `DataList` whose `Err()` is the error recorded when the view failed.

#### Scenario: An invalid window
- **WHEN** `NewDataList(1.0, 2.0).Rolling(RollingOptions{Window: 0}).Mean()`
- **THEN** 結果長度為 0，結果的 `Err()` 非 nil，來源 list 的 `Err()` 也非 nil

#### Scenario: EWM without a decay parameter
- **WHEN** `NewDataList(1.0, 2.0).EWM(EWMOptions{}).Std()`
- **THEN** 結果長度為 0，結果的 `Err()` 非 nil

#### Scenario: A valid window with too few observations is still a list of nil
- **WHEN** `NewDataList(1.0, 2.0).Rolling(RollingOptions{Window: 3}).Mean()`
- **THEN** 結果為 `[nil, nil]`，`Err()` 為 nil

### Requirement: RollingDataList correlation reducers check arguments

`RollingDataList.Apply(nil)` and `Corr`, `Cov` and `Beta` with a `nil` list SHALL return an empty list carrying the error they record.

#### Scenario: Nil argument to correlation reducers
- **WHEN** `RollingDataList` 呼叫 `Apply(nil)`，或 `Corr(nil)`、`Cov(nil)`、`Beta(nil)`
- **THEN** 結果長度為 0 且帶著記錄的錯誤

### Requirement: DataTable column window views record and propagate errors

When `DataTable.RollingCol`, `DataTable.EWMCol` or `DataTable.ExpandingCol` cannot build its view, every reducer SHALL return an empty `DataList` carrying the recorded error, and invalid options SHALL also be recorded on the table.

#### Scenario: A rolling option on a table column
- **WHEN** `dt.RollingCol("A", RollingOptions{Window: 3, MinObs: 5}).Mean()`
- **THEN** 結果長度為 0 並帶著錯誤，`dt.Err()` 也非 nil

#### Scenario: Missing column on expanding view
- **WHEN** `dt.ExpandingCol("missing", 1).Sum()`
- **THEN** 結果長度為 0 並帶著錯誤，`dt.Err()` 也非 nil

### Requirement: Missing column transforms return empty error lists

`DataTable.ShiftCol`, `DiffCol`, `PctChangeCol`, `CumSumCol`, `CumProdCol`, `CumMaxCol` and `CumMinCol` SHALL return an empty list carrying the table's error when the column is not there.

#### Scenario: A column that is not there
- **WHEN** `dt.ShiftCol("missing", 1)`
- **THEN** 回傳長度 0、`Err()` 非 nil 的 list，`dt.Err()` 非 nil

### Requirement: Grouped column transforms propagate group failures

`GroupedColumnTransform.As` SHALL return an empty list carrying the error it records, and when the per-group transform fails for a group, `As` SHALL stop, record that error on the table, and return an empty list carrying it.

#### Scenario: An invalid option inside a grouped transform
- **WHEN** `dt.GroupBy("id").RollingCol("price", RollingOptions{Window: 0}).Mean().As("x")`
- **THEN** 結果長度為 0 並帶著錯誤，而不是整欄 nil，`dt.Err()` 非 nil
