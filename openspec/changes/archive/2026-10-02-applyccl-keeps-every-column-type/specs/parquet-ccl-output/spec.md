## ADDED Requirements

### Requirement: ApplyCCL writes back an unwritten column exactly

`ApplyCCL` SHALL write a column no statement writes back with the type and the values the file gave it, whatever its Arrow type, including integer and float widths other than 64 bits, dates, decimals, large strings, lists and structs, and SHALL NOT fail because the file has such a column. A column a statement assigns SHALL keep its type, `int8` to `int64`, the unsigned widths, `float32`, `float64`, `Date32` and `Date64` included, when that type holds every value written into it exactly.

#### Scenario: A file with columns of many types
- **WHEN** 對一個含 `int32`、`float32`、`Date32`、`decimal128(10, 2)`、`large_string`、`list<int64>` 欄位的檔案執行 `ApplyCCL(ctx, path, "NEW('n') = 1")`
- **THEN** 不回傳錯誤，讀回後這些欄的 Arrow 型別與每一格的值都與原檔相同

#### Scenario: Rows held back by LEAD
- **WHEN** 對同一個檔案執行 `ApplyCCL(ctx, path, "NEW('n') = LEAD(['i32'], 1500)")`
- **THEN** 其他欄的型別與值都與原檔相同

#### Scenario: An int32 column assigned whole numbers
- **WHEN** 對 `int32` 欄 `i32` 執行 `ApplyCCL(ctx, path, "['i32'] = ['i32'] * 2")`
- **THEN** 讀回的 `i32` 仍是 `int32`，值為原來的兩倍
