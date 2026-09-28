# chainable-never-nil Specification

## Purpose
可串接的方法永遠回傳可用的物件，不回傳 nil。

## Requirements

### Requirement: Chainable methods never return nil

`DataList` 上回傳 `*DataList` 的方法 SHALL NOT 回傳 nil。失敗時 SHALL 回傳可用的空 `DataList`，錯誤 SHALL 同時記錄在該結果與接收者上。

#### Scenario: Invalid window keeps the chain alive
- **WHEN** 對 3 個元素的 list 呼叫 `MovingAverage(0).Sort()`
- **THEN** 不 panic，結果長度為 0，接收者的 `Err()` 非 nil

### Requirement: isr keeps its block syntax on failure

`isr` 的 `DT.From`、`Col`、`Row`、`Push`、`UseDL`、`UseDT` 失敗時 SHALL 回傳可繼續串接的物件並記錄 `Err()`，SHALL NOT 結束程序、SHALL NOT 回傳 nil。

#### Scenario: Missing file
- **WHEN** `isr.DT.From(isr.CSV{FilePath: "no_such.csv"})`
- **THEN** 回傳非 nil 的 `*dt`，其 `Err()` 指出讀檔失敗，程序繼續

### Requirement: The rule is enforced by a test, not by review

A test SHALL parse every non-test Go file in the module and fail when a method that returns its receiver's type without an accompanying `error` result returns a literal `nil` in that position. The test SHALL cover the whole module, not only `DataList` and `isr`, and SHALL fail rather than pass when its own traversal or matching finds implausibly little.

#### Scenario: A new method breaks the rule
- **WHEN** 有人新增一個回傳 `*DataList`、沒有 error 回傳值的方法，並在失敗時 `return nil`
- **THEN** 測試失敗並指出該方法的檔名與行號

#### Scenario: The check cannot pass vacuously
- **WHEN** 走訪或比對邏輯壞掉，掃到的檔案或方法數量異常地少
- **THEN** 測試失敗，而不是因為找不到違規而通過

### Requirement: Getters returning a DataList without an error never return nil

A method in `ml` or `nn` that returns an `*insyra.DataList` or `*insyra.DataTable` and no `error` SHALL NOT return nil. When it has nothing to give, it SHALL return an empty, usable value whose `Err()` states the reason. A guard against nil SHALL be kept wherever it can still fire — a value that may come from a third-party implementation of a public interface, or from an in-repo caller that passes nil deliberately — and SHALL be removed only where it has been shown to be unreachable.

#### Scenario: An unfitted classifier is asked for its classes
- **WHEN** 對尚未 fit 的分類器呼叫 `Classes()`
- **THEN** 回傳長度為 0 的 `*insyra.DataList`，其 `Err()` 說明模型尚未 fit，且呼叫任何方法都不會 panic

#### Scenario: A third-party classifier still returns nil
- **WHEN** 函式庫外部的 `ml.Classifier` 實作從自己的 `Classes()` 回傳 nil
- **THEN** insyra 消費該值的位置仍有 nil 防護，且 `ml/mltest` 的一致性檢查會指出這違反協定，而不是自己 panic

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
