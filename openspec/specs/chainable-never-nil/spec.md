# chainable-never-nil Specification

## Purpose
可串接的方法永遠回傳可用的物件，不回傳 nil，失敗時的形狀只有兩種：產生新 list 的方法回傳帶著錯誤的空 list，直接修改接收者的方法回傳原本、內容不變的接收者。這讓一串呼叫中途失敗也不會 nil dereference，也不會把資料丟掉。

## Requirements

### Requirement: Chainable methods never return nil

`DataList` 上回傳 `*DataList` 的方法 SHALL NOT 回傳 nil。會產生新 list 的方法失敗時 SHALL 回傳可用的空 `DataList`，錯誤 SHALL 同時記錄在該結果與接收者上。直接修改接收者的方法失敗時 SHALL 回傳接收者本身、內容不變，錯誤記錄在接收者上。

#### Scenario: Invalid window keeps the chain alive
- **WHEN** 對 3 個元素的 list 呼叫 `Sample(0, false).Sort()`
- **THEN** 不 panic，結果長度為 0，接收者的 `Err()` 非 nil

#### Scenario: A failed in-place method keeps the data
- **WHEN** 對 `[1.0, 2.0, "x"]` 呼叫 `dl = dl.Normalize()`
- **THEN** `dl` 仍是原本的 list，內容為 `[1.0, 2.0, "x"]`，`Err()` 非 nil

#### Scenario: A failed sample carries its error
- **WHEN** 呼叫 `Sample(0, false)`、`Sample(9, false)`（3 個元素，不放回）或 `SampleFrac(2, false)`
- **THEN** 回傳長度 0 的新 list，其 `Err()` 非 nil，接收者的 `Err()` 也非 nil

#### Scenario: Two optional values do not return the receiver
- **WHEN** 呼叫 `Rank(true, false)` 或 `Shift(1, 0, 0)`
- **THEN** 回傳的不是接收者，而是長度 0、`Err()` 非 nil 的新 list，接收者內容不變

#### Scenario: Sampling an empty list is not a failure
- **WHEN** 對空 list 呼叫 `Sample(1, false)` 或 `SampleFrac(0.5, false)`
- **THEN** 回傳空 list，結果與接收者的 `Err()` 皆為 nil

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

When `DataList.Rolling`, `DataList.EWM`, `DataTable.RollingCol`, `DataTable.EWMCol` or `DataTable.ExpandingCol` cannot build its view (an invalid option, or a column that is not there), every reducer of the returned view SHALL return an empty `DataList` whose `Err()` is the error recorded when the view failed, not a list of `nil` the source's length. An invalid option passed to `DataTable.RollingCol` or `EWMCol` SHALL also be recorded on the table.

#### Scenario: An invalid window
- **WHEN** `NewDataList(1.0, 2.0).Rolling(RollingOptions{Window: 0}).Mean()`
- **THEN** 結果長度為 0，結果的 `Err()` 非 nil，來源 list 的 `Err()` 也非 nil

#### Scenario: EWM without a decay parameter
- **WHEN** `NewDataList(1.0, 2.0).EWM(EWMOptions{}).Std()`
- **THEN** 結果長度為 0，結果的 `Err()` 非 nil

#### Scenario: A rolling option on a table column
- **WHEN** `dt.RollingCol("A", RollingOptions{Window: 3, MinObs: 5}).Mean()`
- **THEN** 結果長度為 0 並帶著錯誤，`dt.Err()` 也非 nil

#### Scenario: A valid window with too few observations is still a list of nil
- **WHEN** `NewDataList(1.0, 2.0).Rolling(RollingOptions{Window: 3}).Mean()`
- **THEN** 結果為 `[nil, nil]`，`Err()` 為 nil

### Requirement: A reducer given nothing to work on carries its failure

`RollingDataList.Apply(nil)` and `Corr`, `Cov` and `Beta` with a `nil` list SHALL return an empty list carrying the error they record.

#### Scenario: Apply without a function
- **WHEN** `NewDataList(1.0, 2.0, 3.0).Rolling(RollingOptions{Window: 2}).Apply(nil)`
- **THEN** 結果長度為 0，結果的 `Err()` 非 nil

### Requirement: A column transform on a missing column carries the table's error

`DataTable.ShiftCol`, `DiffCol`, `PctChangeCol`, `CumSumCol`, `CumProdCol`, `CumMaxCol` and `CumMinCol` SHALL return an empty list carrying the table's error when the column is not there.

#### Scenario: A column that is not there
- **WHEN** `dt.ShiftCol(Name("missing"), 1)`，以及 `dt.ExpandingCol(Name("missing"), 1).Sum()`
- **THEN** 兩者都回傳長度 0、`Err()` 非 nil 的 list，`dt.Err()` 非 nil

### Requirement: A grouped transform stops at the first failing group

`GroupedColumnTransform.As` SHALL return an empty list carrying the error it records. When the per-group transform fails for a group, `As` SHALL stop, record that error on the table, and return an empty list carrying it, instead of a column of `nil`.

#### Scenario: An invalid option inside a grouped transform
- **WHEN** `dt.GroupBy(Name("id")).RollingCol(Name("price"), RollingOptions{Window: 0}).Mean().As("x")`
- **THEN** 結果長度為 0 並帶著錯誤，而不是整欄 nil，`dt.Err()` 非 nil
