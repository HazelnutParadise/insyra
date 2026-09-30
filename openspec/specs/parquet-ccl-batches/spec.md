# parquet-ccl-batches Specification

## Purpose
How `FilterWithCCL` and `ApplyCCL` batch a Parquet file: the batch size is fixed at 1,000 rows rather than a setting, because each batch is evaluated on its own, and the documentation says which expressions see only their batch.

## Requirements

### Requirement: The CCL batch is fixed and its effect is documented

`FilterWithCCL` and `ApplyCCL` SHALL read the file in batches of 1,000 rows, a size that is not a setting, because each batch is evaluated on its own and a different size would change the result of an expression that reads beyond the current row. `Docs/parquet.md` SHALL state the batch size, SHALL state that an aggregate, the row index `#` and a reference to a fixed row see only the current batch, SHALL state that a sequence function is not evaluated row by row in these functions, and SHALL name loading the file with `Read` and using the `DataTable` CCL methods as the way to evaluate such an expression over the whole column.

#### Scenario: An aggregate in a filter
- **WHEN** 對 `A` 為 1 到 2,500 的檔案執行 `FilterWithCCL(ctx, path, "A > AVG(A)")`
- **THEN** 保留 1,250 列，第一列的 `A` 為 501，也就是每 1,000 列各自比較自己的平均

#### Scenario: The row index in ApplyCCL
- **WHEN** 對 2,500 列的檔案執行 `ApplyCCL(ctx, path, "NEW('i') = #")` 後讀回
- **THEN** 第 1,000 列的 `i` 為 999，第 1,001 列為 0
