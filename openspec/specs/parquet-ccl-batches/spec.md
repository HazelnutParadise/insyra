# parquet-ccl-batches Specification

## Purpose
How `FilterWithCCL` and `ApplyCCL` read a Parquet file in batches of 1,000 rows and still give the answer the same expression gives on the loaded table: the parts of an expression that read beyond the current row are computed over the whole file first, holding only running totals and the fixed rows named, a built-in sequence function that is a whole expression is streamed across batches, and anything else is computed from the whole of the columns it reads, and only those.

## Requirements

### Requirement: Built-in sequence functions are streamed across batches

`FilterWithCCL` and `ApplyCCL` SHALL compute a built-in sequence function (`LAG`, `LEAD`, `DIFF`, `PCT_CHANGE`, `CUMSUM`, `CUMPROD`, `CUMMAX`, `CUMMIN`, `ROLLING_SUM`, `ROLLING_MEAN`, `ROLLING_MIN`, `ROLLING_MAX`, `ROLLING_STD`) that is the whole filter expression or the whole right-hand side of a `NEW` or an assignment over the whole file, giving value for value what the loaded table gives, while holding only the rows its shift or window reaches across a batch boundary.

#### Scenario: A running sum across batches
- **WHEN** 對 `A` 為 1 到 2,500 的檔案執行 `ApplyCCL(ctx, path, "NEW('c') = CUMSUM(A)")` 後讀回
- **THEN** 第 1,001 列的 `c` 為 501,501，與載入成表格後以 `ExecuteCCL` 執行同一段腳本的結果逐值相同

#### Scenario: A look-ahead across a batch boundary
- **WHEN** 對檔案執行 `ApplyCCL(ctx, path, "NEW('n') = LEAD(A, 3); NEW('m') = ['n'] - A")`
- **THEN** `n` 與 `m` 與載入成表格後以 `ExecuteCCL` 執行同一段腳本的結果逐值相同，最後三列的 `n` 為 nil

#### Scenario: A sequence filter
- **WHEN** 執行 `FilterWithCCL(ctx, path, "DIFF(A)")`
- **THEN** 保留的列與載入成表格後以 `AddColUsingCCL` 算出同一個運算式再篩選的結果相同

### Requirement: A sequence function that cannot stream holds whole columns

A sequence function SHALL stream only when its column argument holds no sequence function and changes from row to row and its other arguments are constants once whole-file parts are computed. A sequence function anywhere else, or one these conditions or a caller's registration keep from streaming, SHALL be computed by holding the whole columns it reads, giving what the loaded table gives, its error included.

#### Scenario: A nested sequence
- **WHEN** 執行 `ApplyCCL(ctx, path, "NEW('c') = LAG(CUMSUM(A), 1)")`
- **THEN** `c` 與載入成表格後以 `ExecuteCCL` 執行同一段腳本的結果逐值相同

#### Scenario: A sequence inside an expression
- **WHEN** 執行 `ApplyCCL(ctx, path, "NEW('c') = CUMSUM(A) + 1")`
- **THEN** 回傳與載入成表格後 `ExecuteCCL` 相同的錯誤（`invalid operands for +`），檔案位元組不變

### Requirement: Batches of a fixed size give the loaded table's answer

`FilterWithCCL` and `ApplyCCL` SHALL read the file in batches of 1,000 rows, a size that is not a setting. The result SHALL equal, value for value, what the same expression or script gives on the file loaded with `Read` and evaluated with the `DataTable` CCL methods, and memory SHALL still hold one batch, plus the fixed rows referenced. The row index `#` SHALL be the row's position in the file.

#### Scenario: The row index in ApplyCCL
- **WHEN** 對 2,500 列的檔案執行 `ApplyCCL(ctx, path, "NEW('i') = #")` 後讀回
- **THEN** 第 1,000 列的 `i` 為 999，第 1,001 列為 1,000

### Requirement: Whole-file parts are computed before the rows

Before evaluating the rows, `FilterWithCCL` and `ApplyCCL` SHALL compute over the whole file every part of the expression that reads beyond the current row and that they can compute in batches: the built-in aggregates `SUM`, `AVG`, `COUNT`, `MIN`, `MAX`, `VAR`, `VARP`, `STDEV` and `STDEVP`, nested or not, and a reference to a fixed row or range of rows.

#### Scenario: An aggregate in a filter
- **WHEN** 對 `A` 為 1 到 2,500 的檔案執行 `FilterWithCCL(ctx, path, "A > AVG(A)")`
- **THEN** 保留 1,250 列，第一列的 `A` 為 1,251，與載入成表格後同一個運算式的結果相同

#### Scenario: A fixed row in a later batch
- **WHEN** 執行 `FilterWithCCL(ctx, path, "A == A.1500")`
- **THEN** 只保留第 1,501 列

### Requirement: Each statement of ApplyCCL sees the ones before it

In `ApplyCCL`, each statement SHALL see the file as the statements before it leave it, including the columns they created or replaced.

#### Scenario: A later statement reads a column an earlier one made
- **WHEN** 對檔案執行 `ApplyCCL(ctx, path, "NEW('c') = A - AVG(A); NEW('d') = ['c'] / SUM(['c'])")`
- **THEN** `c` 與 `d` 與載入成表格後以 `ExecuteCCL` 執行同一段腳本的結果逐值相同

#### Scenario: A later statement reads a column an earlier one replaced
- **WHEN** 對檔案執行 `ApplyCCL(ctx, path, "['A'] = A * 2; NEW('b') = A")`
- **THEN** `b` 等於加倍後的 `A`，與載入成表格後以 `ExecuteCCL` 執行同一段腳本的結果逐值相同

### Requirement: A part that cannot be computed fails only the rows that reach it

A part that cannot be computed, such as `STDEV` over one value or a row past the end, SHALL fail only the rows whose evaluation reaches it, as on the loaded table.

#### Scenario: A part that fails only where a row reaches it
- **WHEN** 對只有一列的檔案執行 `FilterWithCCL(ctx, path, "IF(COUNT(A) >= 2, A > STDEV(A), TRUE)")`
- **THEN** 保留那一列、不回傳錯誤，與載入成表格後的結果相同

### Requirement: What batches cannot compute is computed over whole columns

In `FilterWithCCL` and `ApplyCCL`, an expression or statement with a part that cannot be computed in batches (`MEDIAN`, an aggregate a caller registered, a sequence function that cannot stream, a row reference computed from the current row, or a column range inside an aggregate's argument) SHALL be computed by holding the columns it reads, and only those, in full, giving the answer or the error of the `DataTable` CCL methods; in `ApplyCCL` this SHALL happen before the file is changed.

#### Scenario: A median
- **WHEN** 對有 `A`、`B` 兩欄的檔案執行 `ApplyCCL(ctx, path, "NEW('c') = A - MEDIAN(A)")`
- **THEN** `c` 與載入成表格後以 `ExecuteCCL` 執行同一段腳本的結果逐值相同，且計算時只把 `A` 整欄讀進記憶體

#### Scenario: A row computed from the current one
- **WHEN** 執行 `FilterWithCCL(ctx, path, "IF(# > 0, A > A.(# - 1), FALSE)")`
- **THEN** 保留的列與載入成表格後以 `AddColUsingCCL` 算出同一個運算式再篩選的結果相同

### Requirement: The parquet guide states how CCL reads a file

`Docs/parquet.md` SHALL state the batch size, which parts are computed in batches and how many extra reads they take, and which hold whole columns.

#### Scenario: Reading the guide
- **WHEN** 讀 `Docs/parquet.md` 中 `FilterWithCCL` 與 `ApplyCCL` 的說明
- **THEN** 寫明每批 1,000 列、哪些部分分批計算及各多讀幾次檔案，以及哪些部分會把整欄讀進記憶體

### Requirement: `@` in ApplyCCL holds each row's own values

`ApplyCCL` SHALL give each row's `@` the values of that row, so that a column built from `@` differs from row to row as the file does.

#### Scenario: Each row's current values
- **WHEN** 對 `parquet/ccl_test.go` 的四列測試檔執行 `ApplyCCL(ctx, path, "NEW('r') = @")` 後讀回
- **THEN** `r` 依序為 `[1 10 a true]`、`[2 20 b false]`、`[3 30 c true]`、`[4 40 d false]`，每列都是自己那一列的值
