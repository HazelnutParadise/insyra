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
