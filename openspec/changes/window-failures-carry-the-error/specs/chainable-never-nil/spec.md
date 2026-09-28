## ADDED Requirements

### Requirement: A window view hands its failure to every reducer

When `DataList.Rolling`, `DataList.EWM`, `DataTable.RollingCol`, `DataTable.EWMCol` or `DataTable.ExpandingCol` cannot build its view (an invalid option, or a column that is not there), every reducer of the returned view SHALL return an empty `DataList` whose `Err()` is the error recorded when the view failed. `RollingDataList.Apply(nil)` and `Corr`, `Cov` and `Beta` with a `nil` list SHALL return an empty list carrying the error they record. An invalid option passed to `DataTable.RollingCol` or `EWMCol` SHALL also be recorded on the table. `DataTable.ShiftCol`, `DiffCol`, `PctChangeCol`, `CumSumCol`, `CumProdCol`, `CumMaxCol` and `CumMinCol` SHALL return an empty list carrying the table's error when the column is not there. `GroupedColumnTransform.As` SHALL return an empty list carrying the error it records, and when the per-group transform fails for a group, `As` SHALL stop, record that error on the table, and return an empty list carrying it, instead of a column of `nil`. The failed result SHALL stay empty rather than a list of `nil` the source's length, which a valid window with too few observations also produces.

#### Scenario: An invalid window
- **WHEN** `NewDataList(1.0, 2.0).Rolling(RollingOptions{Window: 0}).Mean()`
- **THEN** 結果長度為 0，結果的 `Err()` 非 nil，來源 list 的 `Err()` 也非 nil

#### Scenario: EWM without a decay parameter
- **WHEN** `NewDataList(1.0, 2.0).EWM(EWMOptions{}).Std()`
- **THEN** 結果長度為 0，結果的 `Err()` 非 nil

#### Scenario: A rolling option on a table column
- **WHEN** `dt.RollingCol("A", RollingOptions{Window: 3, MinObs: 5}).Mean()`
- **THEN** 結果長度為 0 並帶著錯誤，`dt.Err()` 也非 nil

#### Scenario: A column that is not there
- **WHEN** `dt.ShiftCol(Name("missing"), 1)`，以及 `dt.ExpandingCol(Name("missing"), 1).Sum()`
- **THEN** 兩者都回傳長度 0、`Err()` 非 nil 的 list，`dt.Err()` 非 nil

#### Scenario: An invalid option inside a grouped transform
- **WHEN** `dt.GroupBy(Name("id")).RollingCol(Name("price"), RollingOptions{Window: 0}).Mean().As("x")`
- **THEN** 結果長度為 0 並帶著錯誤，而不是整欄 nil，`dt.Err()` 非 nil

#### Scenario: A valid window with too few observations is still a list of nil
- **WHEN** `NewDataList(1.0, 2.0).Rolling(RollingOptions{Window: 3}).Mean()`
- **THEN** 結果為 `[nil, nil]`，`Err()` 為 nil
