# ccl-evaluation-safety Specification

## Purpose
CCL 求值安全契約：使用者運算式不得 panic、不得讓列互相別名、不得越界讀取、時長不得被靜默誤比。
## Requirements
### Requirement: Keywords and out-of-range references

`TRUE`／`FALSE`／`NULL`／`NIL` 不分大小寫 SHALL 是字面值。其餘裸識別字 SHALL 只解析為 Excel 式欄索引，賦值左右兩側同一套規則；欄名 SHALL 只能以 `['name']` 指涉。解析 SHALL NOT 退回欄名表，識別字含數字或底線時 SHALL 回報錯誤而非改查欄名；`[A]` 這種括號索引同一套規則，內容不是欄位字母時 SHALL 在編譯階段失敗。綁定後引用超過最後一欄的 Excel 式索引 SHALL 使 `AddColUsingCCL`／`EditCol*UsingCCL`／`ExecuteCCL` 回報錯誤（`Err()`），SHALL NOT 產生整欄 nil。錯誤訊息 SHALL 在該表存在同名欄位時指出改寫成 `['name']`，該資訊 SHALL NOT 影響解析結果。

#### Scenario: Reference past the last column
- **WHEN** 單欄表執行 `AddColUsingCCL("r", "E + 1")`
- **THEN** `Err()` 非 nil 且表仍是一欄

#### Scenario: A bare word that names a column

- **WHEN** 欄位名為 `price` 的表執行 `AddColUsingCCL("r", "price * 2")`
- **THEN** `Err()` 非 nil，訊息說明 `price` 被讀成欄索引並指出改寫成 `['price']`
- **AND** 同一張表的 `['price'] * 2` 照常求值

#### Scenario: A bare word that cannot be an index at all

- **WHEN** 欄位名為 `qty_1` 的表執行 `AddColUsingCCL("r", "qty_1 * 2")`
- **THEN** `Err()` 非 nil，訊息指出改寫成 `['qty_1']`，SHALL NOT 因為它不是純字母就改查欄名

#### Scenario: A bracketed reference that is not letters

- **WHEN** 求值 `[qty_1] * 2`
- **THEN** 失敗 SHALL 在編譯階段而非第 0 列，訊息指出改寫成 `['qty_1']`

#### Scenario: An assignment target follows the same rule

- **WHEN** 欄位名為 `price` 的表執行 `ExecuteCCL("price = ['price'] * 10")`
- **THEN** `Err()` 非 nil 且訊息指出改寫成 `['price']`
- **AND** `ExecuteCCL("['price'] = ['price'] * 10")` 與 `ExecuteCCL("A = A * 10")` 照常執行

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

### Requirement: Numeric arguments give the same answer on every platform

When CCL turns a numeric argument into an integer or a duration, the result SHALL NOT depend on the platform. NaN, infinities and values a `time.Duration` or a date shift cannot hold SHALL be refused. The exception is a character count, character position or digit count: it SHALL be clamped, so that a count past the end of a string still means "to the end".

#### Scenario: A huge length
- **WHEN** 在 amd64 或 arm64 上求值 `MID('abc', 2, 10^300)`
- **THEN** 兩者都得到 `"bc"`

#### Scenario: A huge date shift
- **WHEN** 求值 `DATEADD(D, 10^300, 'day')` 或 `D + 10^300`
- **THEN** 回傳錯誤，不產生日期

