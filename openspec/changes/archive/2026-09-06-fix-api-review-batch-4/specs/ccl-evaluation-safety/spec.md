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

序列函數作為另一個序列或聚合函數的引數 SHALL 保留整欄；是否當成整欄 SHALL 由引數本身是否為序列函數呼叫決定，SHALL NOT 看結果長度是否剛好等於列數。位移、視窗或重複次數為 NaN、無限大或超出 int64 範圍，或重複結果的長度超出 int 時，SHALL 回錯誤，SHALL NOT panic；比欄位還長的位移或視窗 SHALL 得到 nil，SHALL NOT 回錯誤；字元長度、位置與位數則依「Numeric arguments give the same answer on every platform」截到上限，SHALL NOT 回錯誤。使用者註冊的聚合或序列函數若 panic，SHALL 由呼叫的方法（例如 `AddColUsingCCL`）自身的 recover 處理：方法回傳 nil 並記錄 `Err()`，與 v0.3.2 相同。

#### Scenario: Absurd shift
- **WHEN** 求值 `LEAD(A, 10^300)`
- **THEN** 回傳錯誤且程序不 panic

#### Scenario: A row read on a square table
- **WHEN** 在 3 欄 3 列的表上以已註冊、回傳引數長度的聚合求值 `ZZLEN(@.0)`
- **THEN** 結果為 1，與 2 欄 3 列的表相同

### Requirement: Registry concurrency and NaN aggregation

`RegisterFunction` 與求值並行 SHALL 無 data race。`SUM`／`AVG` SHALL 跳過 NaN，與 `MAX`／`MEDIAN` 一致。`NewMapContext` SHALL 依欄名排序決定 Excel 索引。

#### Scenario: NaN in SUM
- **WHEN** 欄位為 `[10, NaN, 5]` 求 `SUM(A)`
- **THEN** 結果為 15
