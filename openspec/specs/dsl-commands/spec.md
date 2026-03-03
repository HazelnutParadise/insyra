# dsl-commands Specification

## Purpose
TBD - created by archiving change cli-repl. Update Purpose after archive.
## Requirements
### Requirement: Load data from file
系統 SHALL 提供 `load` 命令從檔案載入資料為 DataTable。

#### Scenario: Load CSV
- **WHEN** 使用者執行 `load "data.csv" as t1`
- **THEN** 系統呼叫 `ReadCSV_File()` 載入 CSV 並將結果存為變數 `t1`

#### Scenario: Load Excel with sheet
- **WHEN** 使用者執行 `load "data.xlsx" sheet Sales as t1`
- **THEN** 系統呼叫 `ReadExcelSheet()` 載入指定工作表

#### Scenario: Load JSON
- **WHEN** 使用者執行 `load "data.json" as t1`
- **THEN** 系統呼叫 `ReadJSON_File()` 載入 JSON

#### Scenario: Load Parquet
- **WHEN** 使用者執行 `load parquet "data.parquet" as t1`
- **THEN** 系統呼叫 parquet.Read() 載入 Parquet 檔

#### Scenario: Load without alias
- **WHEN** 使用者執行 `load "data.csv"`（無 `as`）
- **THEN** 結果存入 `$result`

#### Scenario: Auto format detection
- **WHEN** 使用者執行 `load "file.csv"` 或 `load "file.json"`
- **THEN** 系統依副檔名自動判斷格式

### Requirement: Save data to file
系統 SHALL 提供 `save` 命令匯出資料。

#### Scenario: Save as CSV
- **WHEN** 使用者執行 `save t1 "output.csv"`
- **THEN** 系統呼叫 `ToCSV()` 匯出

#### Scenario: Save as JSON
- **WHEN** 使用者執行 `save t1 "output.json"`
- **THEN** 系統呼叫 `ToJSON()` 匯出

#### Scenario: Save as Parquet
- **WHEN** 使用者執行 `save t1 "output.parquet"`
- **THEN** 系統呼叫 parquet.Write() 匯出

### Requirement: Read file preview
系統 SHALL 提供 `read` 命令快速預覽檔案內容（不存入變數）。

#### Scenario: Quick preview
- **WHEN** 使用者執行 `read "data.csv"`
- **THEN** 系統載入並顯示檔案前幾筆資料，不將結果存入環境變數

### Requirement: Convert format
系統 SHALL 提供 `convert` 命令進行格式轉換。

#### Scenario: CSV to Excel
- **WHEN** 使用者執行 `convert "data.csv" "data.xlsx"`
- **THEN** 系統轉換 CSV 為 Excel

### Requirement: Show data
系統 SHALL 提供 `show` 命令顯示資料內容，支援正數（前 N 筆）、負數（後 N 筆）、範圍。

#### Scenario: Show all
- **WHEN** 使用者執行 `show t1`
- **THEN** 系統呼叫 `ShowRange()` 顯示全部

#### Scenario: Show first N
- **WHEN** 使用者執行 `show t1 10`
- **THEN** 系統呼叫 `ShowRange(10)` 顯示前 10 筆

#### Scenario: Show last N
- **WHEN** 使用者執行 `show t1 -5`
- **THEN** 系統呼叫 `ShowRange(-5)` 顯示後 5 筆

#### Scenario: Show range
- **WHEN** 使用者執行 `show t1 2 10`
- **THEN** 系統呼叫 `ShowRange(2, 10)` 顯示第 2~9 筆

#### Scenario: Show from index to end
- **WHEN** 使用者執行 `show t1 5 _`
- **THEN** 系統呼叫 `ShowRange(5, nil)` 顯示第 5 筆到結尾（`_` 代表 nil）

### Requirement: Show types
系統 SHALL 提供 `types` 命令顯示各元素型別。

#### Scenario: Show types
- **WHEN** 使用者執行 `types t1`
- **THEN** 系統呼叫 `ShowTypes()` 顯示欄位型別資訊

### Requirement: Summary statistics
系統 SHALL 提供 `summary` 命令顯示統計摘要。

#### Scenario: DataTable summary
- **WHEN** 使用者執行 `summary t1`
- **THEN** 系統呼叫 `t1.Summary()` 顯示統計摘要

### Requirement: Shape display
系統 SHALL 提供 `shape` 命令顯示資料維度。

#### Scenario: DataTable shape
- **WHEN** 使用者執行 `shape t1`
- **THEN** 系統顯示 `(rows, cols)` 維度數字

### Requirement: Column and row names
系統 SHALL 提供 `cols` 和 `rows` 命令列出欄名和列名。

#### Scenario: List column names
- **WHEN** 使用者執行 `cols t1`
- **THEN** 系統呼叫 `ColNames()` 顯示所有欄位名稱

#### Scenario: List row names
- **WHEN** 使用者執行 `rows t1`
- **THEN** 系統呼叫 `RowNames()` 顯示所有列名稱

### Requirement: Column and row extraction
系統 SHALL 提供 `col` 和 `row` 命令取出欄或列。

#### Scenario: Extract column by name
- **WHEN** 使用者執行 `col t1 "revenue" as d1`
- **THEN** 系統取出名為 "revenue" 的欄位，存為 DataList 變數 `d1`

#### Scenario: Extract row by index
- **WHEN** 使用者執行 `row t1 5 as r1`
- **THEN** 系統取出第 5 列，存為 DataList 變數 `r1`

### Requirement: Element access
系統 SHALL 提供 `get` 和 `set` 命令存取單一元素。

#### Scenario: Get element
- **WHEN** 使用者執行 `get t1 3 2`
- **THEN** 系統顯示第 3 列第 2 欄的元素

#### Scenario: Set element
- **WHEN** 使用者執行 `set t1 3 2 100`
- **THEN** 系統更新第 3 列第 2 欄為 100

### Requirement: Add and drop columns/rows
系統 SHALL 提供 `addcol`、`addrow`、`dropcol`、`droprow` 命令。

#### Scenario: Drop column by name
- **WHEN** 使用者執行 `dropcol t1 "temp_col"`
- **THEN** 系統刪除名為 "temp_col" 的欄位

#### Scenario: Drop row by index
- **WHEN** 使用者執行 `droprow t1 0 1 2`
- **THEN** 系統刪除第 0、1、2 列

### Requirement: Set column and row names
系統 SHALL 提供 `setcolnames` 和 `setrownames` 命令。

#### Scenario: Set column names
- **WHEN** 使用者執行 `setcolnames t1 "Name" "Age" "Score"`
- **THEN** 系統設定 t1 的欄名為 Name、Age、Score

### Requirement: Swap columns and rows
系統 SHALL 提供 `swap` 命令交換欄或列。

#### Scenario: Swap columns
- **WHEN** 使用者執行 `swap t1 col "A" "B"`
- **THEN** 系統交換名為 A 和 B 的欄位

#### Scenario: Swap rows
- **WHEN** 使用者執行 `swap t1 row 0 5`
- **THEN** 系統交換第 0 和第 5 列

### Requirement: Transpose
系統 SHALL 提供 `transpose` 命令轉置 DataTable。

#### Scenario: Transpose table
- **WHEN** 使用者執行 `transpose t1 as t2`
- **THEN** 系統轉置 t1 並存為 t2

### Requirement: Filter data
系統 SHALL 提供 `filter` 命令用 CCL 表達式篩選。

#### Scenario: Filter with expression
- **WHEN** 使用者執行 `filter t1 "revenue > 1000" as t_high`
- **THEN** 系統用 CCL 表達式篩選並存結果

### Requirement: Sort data
系統 SHALL 提供 `sort` 命令排序。

#### Scenario: Sort ascending
- **WHEN** 使用者執行 `sort t1 "revenue" asc`
- **THEN** 系統依 revenue 欄升序排序

#### Scenario: Sort descending
- **WHEN** 使用者執行 `sort t1 "revenue" desc`
- **THEN** 系統依 revenue 欄降序排序

### Requirement: Find values
系統 SHALL 提供 `find` 命令搜尋。

#### Scenario: Find rows containing value
- **WHEN** 使用者執行 `find t1 "error"`
- **THEN** 系統顯示所有包含 "error" 的列

### Requirement: Random sample
系統 SHALL 提供 `sample` 命令隨機抽樣。

#### Scenario: Sample N rows
- **WHEN** 使用者執行 `sample t1 100 as t_sample`
- **THEN** 系統從 t1 隨機抽取 100 列存為 t_sample

### Requirement: Clone variable
系統 SHALL 提供 `clone` 命令深複製。

#### Scenario: Clone DataTable
- **WHEN** 使用者執行 `clone t1 as t2`
- **THEN** 系統深複製 t1 為獨立的 t2

### Requirement: Replace values
系統 SHALL 提供 `replace` 命令取代值。

#### Scenario: Replace specific value
- **WHEN** 使用者執行 `replace t1 "N/A" 0`
- **THEN** 系統將所有 "N/A" 取代為 0

#### Scenario: Replace NaN
- **WHEN** 使用者執行 `replace t1 nan 0`
- **THEN** 系統將所有 NaN 取代為 0

#### Scenario: Replace nil
- **WHEN** 使用者執行 `replace t1 nil 0`
- **THEN** 系統將所有 nil 取代為 0

### Requirement: Clean data
系統 SHALL 提供 `clean` 命令移除特殊值。

#### Scenario: Clean NaN
- **WHEN** 使用者執行 `clean t1 nan`
- **THEN** 系統移除包含 NaN 的列

#### Scenario: Clean outliers
- **WHEN** 使用者執行 `clean d1 outliers 2`
- **THEN** 系統移除超過 2 個標準差的離群值

### Requirement: Create DataList manually
系統 SHALL 提供 `newdl` 命令手動建立 DataList。

#### Scenario: Create with values
- **WHEN** 使用者執行 `newdl 1 2 3 4 5 as d1`
- **THEN** 系統建立包含 [1, 2, 3, 4, 5] 的 DataList 並存為 d1

#### Scenario: Create with strings
- **WHEN** 使用者執行 `newdl "apple" "banana" as fruits`
- **THEN** 系統建立包含字串的 DataList

#### Scenario: Create empty
- **WHEN** 使用者執行 `newdl as d1`
- **THEN** 系統建立空的 DataList

### Requirement: Create DataTable manually
系統 SHALL 提供 `newdt` 命令從 DataList 組成 DataTable。

#### Scenario: Create from DataLists
- **WHEN** 使用者執行 `newdt d1 d2 d3 as t1`
- **THEN** 系統以 d1、d2、d3 作為欄位建立 DataTable

#### Scenario: Create empty
- **WHEN** 使用者執行 `newdt as t1`
- **THEN** 系統建立空的 DataTable

### Requirement: DataList statistics commands
系統 SHALL 提供 DataList 統計命令：`sum`、`mean`、`median`、`mode`、`stdev`、`var`、`min`、`max`、`range`、`quartile`、`iqr`、`percentile`、`count`、`counter`。

#### Scenario: Mean of DataList
- **WHEN** 使用者執行 `mean d1`
- **THEN** 系統呼叫 `d1.Mean()` 並顯示結果

#### Scenario: Quartile
- **WHEN** 使用者執行 `quartile d1 1`
- **THEN** 系統呼叫 `d1.Quartile(1)` 顯示第 1 四分位數

#### Scenario: Counter
- **WHEN** 使用者執行 `counter d1`
- **THEN** 系統呼叫 `d1.Counter()` 顯示頻率表

### Requirement: DataList transformation commands
系統 SHALL 提供 DataList 轉換命令：`rank`、`normalize`、`standardize`、`reverse`、`upper`、`lower`、`capitalize`、`parsenums`、`parsestrings`。

#### Scenario: Normalize DataList
- **WHEN** 使用者執行 `normalize d1 as d1_norm`
- **THEN** 系統呼叫 `d1.Normalize()` 並存為 d1_norm

#### Scenario: Reverse DataList
- **WHEN** 使用者執行 `reverse d1`
- **THEN** 系統呼叫 `d1.Reverse()` 原地反轉

### Requirement: Time series commands
系統 SHALL 提供時間序列命令：`movavg`、`expsmooth`、`diff`、`fillnan`。

#### Scenario: Moving average
- **WHEN** 使用者執行 `movavg d1 5 as d1_ma`
- **THEN** 系統呼叫 `d1.MovingAverage(5)` 並存結果

#### Scenario: Fill NaN with mean
- **WHEN** 使用者執行 `fillnan d1 mean`
- **THEN** 系統呼叫 `d1.FillNaNWithMean()`

### Requirement: Advanced statistics commands
系統 SHALL 提供進階統計命令：`corr`、`corrmatrix`、`regression`、`ttest`、`ztest`、`anova`、`ftest`、`chisq`、`pca`、`skewness`、`kurtosis`、`cov`。

#### Scenario: Correlation
- **WHEN** 使用者執行 `corr d1 d2 pearson`
- **THEN** 系統呼叫 `stats.Correlation()` 並顯示結果

#### Scenario: Linear regression
- **WHEN** 使用者執行 `regression linear y x1 x2`
- **THEN** 系統呼叫 `stats.LinearRegression()` 並顯示迴歸結果

#### Scenario: PCA
- **WHEN** 使用者執行 `pca t1 3`
- **THEN** 系統呼叫 `stats.PCA()` 取前 3 個主成份

### Requirement: CCL commands
系統 SHALL 提供 `ccl` 和 `addcolccl` 命令執行 CCL 表達式。

#### Scenario: Execute CCL
- **WHEN** 使用者執行 `ccl t1 "C = A + B"`
- **THEN** 系統呼叫 `ExecuteCCL()` 執行表達式

#### Scenario: Add column with CCL
- **WHEN** 使用者執行 `addcolccl t1 "total" "A + B"`
- **THEN** 系統呼叫 `AddColUsingCCL("total", "A + B")`

### Requirement: Merge tables
系統 SHALL 提供 `merge` 命令合併表格。

#### Scenario: Horizontal inner merge
- **WHEN** 使用者執行 `merge t1 t2 horizontal inner on "id" as t3`
- **THEN** 系統呼叫 `t1.Merge()` 以 id 欄做內合併

### Requirement: Plot visualization
系統 SHALL 提供 `plot` 命令繪圖。

#### Scenario: Line chart
- **WHEN** 使用者執行 `plot line d1 save "chart.png"`
- **THEN** 系統繪製折線圖並存檔

### Requirement: Fetch external data
系統 SHALL 提供 `fetch` 命令擷取外部資料。

#### Scenario: Yahoo Finance history
- **WHEN** 使用者執行 `fetch yahoo AAPL history as t1`
- **THEN** 系統呼叫 Yahoo Finance API 取得歷史價格存為 t1

### Requirement: Variable management
系統 SHALL 提供 `vars`、`drop`、`rename` 命令管理變數。

#### Scenario: List variables
- **WHEN** 使用者執行 `vars`
- **THEN** 系統列出所有變數名稱、型別和摘要資訊

#### Scenario: Drop variable
- **WHEN** 使用者執行 `drop t1`
- **THEN** 系統從變數表中刪除 t1

#### Scenario: Rename variable
- **WHEN** 使用者執行 `rename t1 sales`
- **THEN** 系統將變數 t1 重命名為 sales

