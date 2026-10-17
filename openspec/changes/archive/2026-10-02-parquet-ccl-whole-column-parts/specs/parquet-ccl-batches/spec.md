## REMOVED Requirements

### Requirement: The CCL batch is fixed and its effect is documented
**Reason**: Replaced by "Batches of a fixed size give the loaded table's answer", which computes the parts this requirement refused instead of refusing them.
**Migration**: None; an expression that was refused now gives the loaded table's answer or error.

## ADDED Requirements

### Requirement: Batches of a fixed size give the loaded table's answer

`FilterWithCCL` and `ApplyCCL` SHALL read the file in batches of 1,000 rows, a size that is not a setting. Before evaluating the rows they SHALL compute over the whole file every part of the expression that reads beyond the current row and that they can compute in batches: the built-in aggregates `SUM`, `AVG`, `COUNT`, `MIN`, `MAX`, `VAR`, `VARP`, `STDEV` and `STDEVP`, nested or not, and a reference to a fixed row or range of rows. The row index `#` SHALL be the row's position in the file. In `ApplyCCL`, each statement SHALL see the file as the statements before it leave it, including the columns they created or replaced. The result SHALL equal, value for value, what the same expression or script gives on the file loaded with `Read` and evaluated with the `DataTable` CCL methods, and memory SHALL still hold one batch, plus the fixed rows referenced. A part that cannot be computed, such as `STDEV` over one value or a row past the end, SHALL fail only the rows whose evaluation reaches it, as on the loaded table. An expression or statement holding a part they cannot compute in batches (`MEDIAN`, an aggregate a caller registered, a sequence function other than as the requirement on streamed sequence functions allows, a reference to a row computed from the current one, or a column range inside an expression an aggregate is computed over) SHALL be computed by holding the whole of the columns it reads, and only those, and evaluating it as the `DataTable` CCL methods do, so that its answer or its error is theirs; in `ApplyCCL` this SHALL happen before the file is changed. `Docs/parquet.md` SHALL state the batch size, which parts are computed in batches and how many extra reads they take, and which hold whole columns.

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

#### Scenario: A median
- **WHEN** 對有 `A`、`B` 兩欄的檔案執行 `ApplyCCL(ctx, path, "NEW('c') = A - MEDIAN(A)")`
- **THEN** `c` 與載入成表格後以 `ExecuteCCL` 執行同一段腳本的結果逐值相同，且計算時只把 `A` 整欄讀進記憶體

#### Scenario: A row computed from the current one
- **WHEN** 執行 `FilterWithCCL(ctx, path, "IF(# > 0, A > A.(# - 1), FALSE)")`
- **THEN** 保留的列與載入成表格後以 `AddColUsingCCL` 算出同一個運算式再篩選的結果相同

## MODIFIED Requirements

### Requirement: Built-in sequence functions are streamed across batches

`FilterWithCCL` and `ApplyCCL` SHALL compute a built-in sequence function (`LAG`, `LEAD`, `DIFF`, `PCT_CHANGE`, `CUMSUM`, `CUMPROD`, `CUMMAX`, `CUMMIN`, `ROLLING_SUM`, `ROLLING_MEAN`, `ROLLING_MIN`, `ROLLING_MAX`, `ROLLING_STD`) that is the whole filter expression or the whole right-hand side of a `NEW` or an assignment over the whole file, giving value for value what the loaded table gives, while holding only the rows its shift or window reaches across a batch boundary, when its column argument holds no sequence function and changes from row to row and its other arguments are constants once whole-file parts are computed. A sequence function anywhere else, or one these conditions or a caller's registration keep from streaming, SHALL be computed by holding the whole columns it reads, giving what the loaded table gives, its error included.

#### Scenario: A running sum across batches
- **WHEN** 對 `A` 為 1 到 2,500 的檔案執行 `ApplyCCL(ctx, path, "NEW('c') = CUMSUM(A)")` 後讀回
- **THEN** 第 1,001 列的 `c` 為 501,501，與載入成表格後以 `ExecuteCCL` 執行同一段腳本的結果逐值相同

#### Scenario: A look-ahead across a batch boundary
- **WHEN** 對檔案執行 `ApplyCCL(ctx, path, "NEW('n') = LEAD(A, 3); NEW('m') = ['n'] - A")`
- **THEN** `n` 與 `m` 與載入成表格後以 `ExecuteCCL` 執行同一段腳本的結果逐值相同，最後三列的 `n` 為 nil

#### Scenario: A sequence filter
- **WHEN** 執行 `FilterWithCCL(ctx, path, "DIFF(A)")`
- **THEN** 保留的列與載入成表格後以 `AddColUsingCCL` 算出同一個運算式再篩選的結果相同

#### Scenario: A nested sequence
- **WHEN** 執行 `ApplyCCL(ctx, path, "NEW('c') = LAG(CUMSUM(A), 1)")`
- **THEN** `c` 與載入成表格後以 `ExecuteCCL` 執行同一段腳本的結果逐值相同

#### Scenario: A sequence inside an expression
- **WHEN** 執行 `ApplyCCL(ctx, path, "NEW('c') = CUMSUM(A) + 1")`
- **THEN** 回傳與載入成表格後 `ExecuteCCL` 相同的錯誤（`invalid operands for +`），檔案位元組不變
