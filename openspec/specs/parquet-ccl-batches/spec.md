# parquet-ccl-batches Specification

## Purpose
How `FilterWithCCL` and `ApplyCCL` read a Parquet file in batches of 1,000 rows and still give the answer the same expression gives on the loaded table: the parts of an expression that read beyond the current row are computed over the whole file first, holding only running totals and the fixed rows named, and what cannot be computed that way is refused before anything is read.

## Requirements

### Requirement: The CCL batch is fixed and its effect is documented

`FilterWithCCL` and `ApplyCCL` SHALL read the file in batches of 1,000 rows, a size that is not a setting. Before evaluating the rows they SHALL compute over the whole file every part of the expression that reads beyond the current row and that they support: the built-in aggregates `SUM`, `AVG`, `COUNT`, `MIN`, `MAX`, `VAR`, `VARP`, `STDEV` and `STDEVP`, nested or not, and a reference to a fixed row or range of rows. The row index `#` SHALL be the row's position in the file. In `ApplyCCL`, each statement SHALL see the file as the statements before it leave it, including the columns they created or replaced. The result SHALL equal, value for value, what the same expression or script gives on the file loaded with `Read` and evaluated with the `DataTable` CCL methods, and memory SHALL still hold one batch, plus the fixed rows referenced. A part that cannot be computed, such as `STDEV` over one value or a row past the end, SHALL fail only the rows whose evaluation reaches it, as on the loaded table. An expression holding a part they do not support yet (a sequence function, `MEDIAN`, an aggregate or sequence function a caller registered, a reference to a row computed from the current one, or a column range inside an expression an aggregate is computed over) SHALL be refused with an error naming that part, before the file is changed, whether or not the file has rows. `Docs/parquet.md` SHALL state the batch size, which parts are computed over the whole file and how many extra reads they take, and which are refused.

#### Scenario: An aggregate in a filter
- **WHEN** 對 `A` 為 1 到 2,500 的檔案執行 `FilterWithCCL(ctx, path, "A > AVG(A)")`
- **THEN** 保留 1,250 列，第一列的 `A` 為 1,251，與載入成表格後同一個運算式的結果相同

#### Scenario: The row index in ApplyCCL
- **WHEN** 對 2,500 列的檔案執行 `ApplyCCL(ctx, path, "NEW('i') = #")` 後讀回
- **THEN** 第 1,000 列的 `i` 為 999，第 1,001 列為 1,000

#### Scenario: A later statement reads a column an earlier one made
- **WHEN** 對檔案執行 `ApplyCCL(ctx, path, "NEW('c') = A - AVG(A); NEW('d') = ['c'] / SUM(['c'])")`
- **THEN** `c` 與 `d` 與載入成表格後以 `ExecuteCCL` 執行同一段腳本的結果逐值相同

#### Scenario: A later statement reads a column an earlier one replaced
- **WHEN** 對檔案執行 `ApplyCCL(ctx, path, "['A'] = A * 2; NEW('b') = A")`
- **THEN** `b` 等於加倍後的 `A`，與載入成表格後以 `ExecuteCCL` 執行同一段腳本的結果逐值相同

#### Scenario: A fixed row in a later batch
- **WHEN** 執行 `FilterWithCCL(ctx, path, "A == A.1500")`
- **THEN** 只保留第 1,501 列

#### Scenario: A part that fails only where a row reaches it
- **WHEN** 對只有一列的檔案執行 `FilterWithCCL(ctx, path, "IF(COUNT(A) >= 2, A > STDEV(A), TRUE)")`
- **THEN** 保留那一列、不回傳錯誤，與載入成表格後的結果相同

#### Scenario: A part not supported yet
- **WHEN** 執行 `ApplyCCL(ctx, path, "NEW('c') = CUMSUM(A)")`
- **THEN** 回傳指出 `CUMSUM` 的錯誤，檔案位元組不變
