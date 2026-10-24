## ADDED Requirements

### Requirement: The CCL batch is fixed and each batch is evaluated on its own

`FilterWithCCL` and `ApplyCCL` SHALL read the file in batches of 1,000 rows, a size that is not a setting, and each batch is evaluated on its own.

#### Scenario: An aggregate in a filter
- **WHEN** 對 `A` 為 1 到 2,500 的檔案執行 `FilterWithCCL(ctx, path, "A > AVG(A)")`
- **THEN** 保留 1,250 列，第一列的 `A` 為 501，也就是每 1,000 列各自比較自己的平均（500.5、1,500.5、2,250.5）

#### Scenario: The row index restarts in each batch
- **WHEN** 對 2,500 列的檔案執行 `ApplyCCL(ctx, path, "NEW('i') = #")` 後讀回
- **THEN** 第 1,000 列的 `i` 為 999，第 1,001 列為 0

#### Scenario: A filter on the row index
- **WHEN** 對 2,500 列、`A` 為 1 到 2,500 的檔案執行 `FilterWithCCL(ctx, path, "# == 0")`
- **THEN** 保留第 1、1,001、2,001 列
