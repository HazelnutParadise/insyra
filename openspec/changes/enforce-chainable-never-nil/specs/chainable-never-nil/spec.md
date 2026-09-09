## ADDED Requirements

### Requirement: The rule is enforced by a test, not by review

A test SHALL parse every non-test Go file in the module and fail when a method that returns its receiver's type without an accompanying `error` result returns a literal `nil` in that position. The test SHALL cover the whole module, not only `DataList` and `isr`, and SHALL fail rather than pass when its own traversal or matching finds implausibly little.

#### Scenario: A new method breaks the rule
- **WHEN** 有人新增一個回傳 `*DataList`、沒有 error 回傳值的方法，並在失敗時 `return nil`
- **THEN** 測試失敗並指出該方法的檔名與行號

#### Scenario: The check cannot pass vacuously
- **WHEN** 走訪或比對邏輯壞掉，掃到的檔案或方法數量異常地少
- **THEN** 測試失敗，而不是因為找不到違規而通過
