# ccl-evaluation-safety Specification

## Purpose
CCL 求值安全契約：使用者運算式不得 panic、不得讓列互相別名、時長不得被靜默誤比，函數註冊與求值可以並行。

## Requirements
### Requirement: Keywords are case-insensitive literals

`TRUE`／`FALSE`／`NULL`／`NIL` 不分大小寫 SHALL 是字面值，SHALL NOT 被當成欄位參照。

#### Scenario: Upper-case keywords
- **WHEN** 求值 `IF(TRUE, NULL, 1)`
- **THEN** 結果為 nil

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

### Requirement: Registry concurrency and deterministic MapContext

`RegisterFunction` 與求值並行 SHALL 無 data race。`NewMapContext` SHALL 依欄名排序決定 Excel 索引。

#### Scenario: Map iteration order
- **WHEN** 以欄名 `z`、`a`、`m` 建立 `MapContext`
- **THEN** 欄序固定為 `a`、`m`、`z`
