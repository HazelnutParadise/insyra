## ADDED Requirements

### Requirement: ApplyCCL loses no value to a column's type

`ApplyCCL` SHALL write a column a statement assigns with the type the file gave it when every value written into it can be held by that type without loss, and otherwise, like a column a script creates, with the type `Write` gives a table column holding those values, settled from every value the column receives, not from the first batch. Every column a statement writes SHALL be nullable, so a missing value reads back as `nil`. A value that cannot be written, or a column type `ApplyCCL` cannot write back, SHALL be an error that leaves the file as it was, never a panic.

#### Scenario: A fraction in an integer column
- **WHEN** 對 int64 欄 `num` 為 `[1, 2, 3, 4]` 的檔案執行 `ApplyCCL(ctx, path, "['num'] = ['num'] * 1.5")`
- **THEN** 讀回的 `num` 是 float64 的 `[1.5, 3, 4.5, 6]`，不是截斷後的整數

#### Scenario: Whole numbers keep an integer column
- **WHEN** 對只有一個 int64 欄 `n`、值為 1 到 5 的檔案執行 `ApplyCCL(ctx, path, "['n'] = A * 2")`
- **THEN** 讀回的 `n` 仍是 int64 的 `[2, 4, 6, 8, 10]`

#### Scenario: A created column that starts with missing values
- **WHEN** 對 `A` 為 1 到 2,500、以 `WriteOptions{RowGroupSize: 1000}` 寫出的檔案執行 `ApplyCCL(ctx, path, "NEW('r') = ROLLING_MEAN(A, 1500)")`
- **THEN** 讀回的 `r` 前 1,499 列為 nil、其後是 float64，不是文字

#### Scenario: A missing value in a created column
- **WHEN** 對 `A` 為 `[1, nil, 3]` 的檔案執行 `ApplyCCL(ctx, path, "NEW('c') = A")`
- **THEN** 讀回的 `c` 是 `[1, nil, 3]`
