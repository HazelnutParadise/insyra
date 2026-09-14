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

序列函數作為另一個序列或聚合函數的引數 SHALL 保留整欄。位移、視窗或重複次數為 NaN、無限大或超出 int64 範圍，或重複結果的長度超出 int 時，SHALL 回錯誤，SHALL NOT panic；比欄位還長的位移或視窗 SHALL 得到 nil，SHALL NOT 回錯誤；字元長度、位置與位數則依「Numeric arguments give the same answer on every platform」截到上限，SHALL NOT 回錯誤。使用者註冊的聚合或序列函數若 panic，SHALL 由呼叫的方法（例如 `AddColUsingCCL`）自身的 recover 處理：方法回傳 nil 並記錄 `Err()`，與 v0.3.2 相同。

#### Scenario: Absurd shift
- **WHEN** 求值 `LEAD(A, 10^300)`
- **THEN** 回傳錯誤且程序不 panic

### Requirement: Registry concurrency and deterministic MapContext

`RegisterFunction` 與求值並行 SHALL 無 data race。`NewMapContext` SHALL 依欄名排序決定 Excel 索引。

#### Scenario: Map iteration order
- **WHEN** 以欄名 `z`、`a`、`m` 建立 `MapContext`
- **THEN** 欄序固定為 `a`、`m`、`z`

### Requirement: Numeric arguments give the same answer on every platform

When CCL turns a numeric argument into an integer or a duration, the result SHALL NOT depend on the platform. NaN, infinities, a count or shift outside the int64 range, and a value that overflows a `time.Duration` SHALL be refused; a row range bound SHALL be refused outside the int32 range. A character count, character position or digit count SHALL instead be clamped, so a count past the end of a string still means "to the end". Ordinary values SHALL give the results they gave before.

#### Scenario: A huge length
- **WHEN** 在 amd64 或 arm64 上求值 `MID('abc', 2, 10^300)`
- **THEN** 兩者都得到 `"bc"`

#### Scenario: A huge date shift
- **WHEN** 求值 `DATEADD(D, 10^300, 'day')` 或 `D + 10^300`
- **THEN** 回傳錯誤，不產生日期

#### Scenario: A fractional day count
- **WHEN** 求值 `D + 0.5` 或 `D + 0.01`
- **THEN** 日期分別移動 12 小時與不移動，與原本相同

#### Scenario: A large but deterministic argument
- **WHEN** 求值 `LAG(A, 3000000000)`、`LEN(REPEAT('', 100000000))` 或 `DATEADD(D, 3000000000, 'day')`
- **THEN** 分別得到整欄 nil、`0` 與一個日期，不回傳錯誤
