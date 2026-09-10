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
