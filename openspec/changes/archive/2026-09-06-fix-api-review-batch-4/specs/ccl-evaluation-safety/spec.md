## ADDED Requirements

### Requirement: Keywords and out-of-range references

`TRUE`／`FALSE`／`NULL`／`NIL` 不分大小寫 SHALL 是字面值。綁定後引用超過最後一欄的 Excel 式索引 SHALL 使 `AddColUsingCCL`／`EditCol*UsingCCL`／`ExecuteCCL` 回報錯誤（`Err()`），SHALL NOT 產生整欄 nil。

#### Scenario: Reference past the last column
- **WHEN** 單欄表執行 `AddColUsingCCL("r", "E + 1")`
- **THEN** `Err()` 非 nil 且表仍是一欄

### Requirement: Row snapshot and duration arithmetic

`@` 當值使用時每列 SHALL 得到獨立複本。`time.Duration` 在數值情境 SHALL 換算為秒。

#### Scenario: `@` column
- **WHEN** 三列表執行 `AddColUsingCCL("r", "@")`
- **THEN** 第 i 列的值等於第 i 列本身，三列互不相同

### Requirement: Nested sequence functions and bounded arguments

序列函數作為另一個序列或聚合函數的引數 SHALL 保留整欄。位移、視窗、長度、重複次數超出合理範圍 SHALL 回錯誤，SHALL NOT panic；聚合與序列函數的 panic SHALL 被轉成錯誤。

#### Scenario: Absurd shift
- **WHEN** 求值 `LEAD(A, 10^300)`
- **THEN** 回傳錯誤且程序不 panic

### Requirement: Registry concurrency and NaN aggregation

`RegisterFunction` 與求值並行 SHALL 無 data race。`SUM`／`AVG` SHALL 跳過 NaN，與 `MAX`／`MEDIAN` 一致。`NewMapContext` SHALL 依欄名排序決定 Excel 索引。

#### Scenario: NaN in SUM
- **WHEN** 欄位為 `[10, NaN, 5]` 求 `SUM(A)`
- **THEN** 結果為 15
