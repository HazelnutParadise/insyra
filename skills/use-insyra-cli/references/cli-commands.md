# Insyra CLI Commands (Complete)

This list is generated from `insyra help` in this repository state.

## Core / Session
- `help` - Show command help
- `version` - Show insyra version
- `exit` - Exit REPL
- `history` - Show command history
- `clear` - Clear terminal screen
- `config` - Read or update global CLI config
- `run` - Run DSL script file

## Environment
- `env` - Environment management
  - `env create <name>`
  - `env list`
  - `env open <name>`
  - `env clear [name] [--keep-history]`
  - `env export [name] <file>`
  - `env import <file> [name] [--force]`
  - `env delete <name>`
  - `env rename <old> <new>`
  - `env info [name]`
- `vars` - List variables in current environment
- `drop` - Delete variable
- `clone` - Deep clone DataTable/DataList variable
- `rename` - Rename variable
- `shape` - Show shape of DataTable/DataList
- `types` - Show value types of DataTable/DataList
- `show` - Display data with optional range (supports negative and `_`)
- `summary` - Show summary statistics

## Data Creation / IO
- `newdl` - Create DataList manually
- `newdt` - Create DataTable from DataList variables
- `load` - Load data into a DataTable variable from a file, parquet, or SQL connection
  - File options: `headers true|false` (default `true`), `rownames true|false` (default `false`), `encoding <enc>` (CSV only), `sheet <name>` (Excel)
- `read` - Quick preview a file without saving variable (forwards the same file options as `load`)
- `save` - Save a DataTable variable to a file or SQL connection
  - File options: `headers true|false` (default `true`), `rownames true|false` (default `false`), `bom true|false` (default `false`, CSV only)
- `convert` - Convert file formats (csv<->xlsx)

## Database (sqlite / mysql / postgres, pure-Go drivers)

- `db connect <name> <dsn>` - Open and register a named connection
- `db list` - List active connections (passwords masked)
- `db tables <name> [schema <s>]` - List tables on a connection
- `db disconnect <name>` - Close and unregister a connection
- `load sql <conn> <table> [...]` - Load a table into a DataTable
- `load sql <conn> query "<SQL>" [params <v1> ...]` - Load a parameterized query result
- `save <var> sql <conn> <table> [...]` - Write a DataTable to a SQL table

## DataTable Structure & Access
- `addcol` - Add one column to DataTable
- `addrow` - Add one row to DataTable
- `dropcol` - Drop columns by name or index
- `droprow` - Drop rows by index or name
- `swap` - Swap DataTable columns or rows
- `transpose` - Transpose DataTable
- `rows` - List DataTable row names
- `cols` - List DataTable column names
- `row` - Extract DataTable row as DataList
- `col` - Extract DataTable column as DataList
- `get` - Get single element from DataTable
- `set` - Set single element in DataTable
- `setrownames` - Set DataTable row names
- `setcolnames` - Set DataTable column names

## Data Processing
- `filter` - Filter DataTable by CCL expression
- `sort` - Sort DataTable by one column
- `sample` - Simple random sample from DataTable
- `find` - Find rows containing value
- `replace` - Replace values in DataTable/DataList
- `clean` - Clean values from DataTable/DataList
- `merge` - Merge two DataTables
- `groupby` - Group a DataTable and aggregate columns (split-apply-combine)
- `pivot` - Reshape long-form DataTable to wide form (long -> wide)
- `unpivot` - Reshape wide-form DataTable to long form (wide -> long)
- `ccl` - Execute CCL statements on DataTable
- `addcolccl` - Add DataTable column using CCL

## DataList Statistics
- `sum` - DataList sum
- `mean` - DataList mean
- `median` - DataList median
- `mode` - DataList mode
- `stdev` - DataList standard deviation
- `var` - DataList variance
- `min` - DataList minimum
- `max` - DataList maximum
- `range` - DataList range
- `quartile` - DataList quartile
- `iqr` - DataList IQR
- `percentile` - DataList percentile
- `count` - Count occurrences
- `counter` - DataList frequency map
- `corr` - Correlation between two DataLists
- `cov` - Covariance between two DataLists
- `corrmatrix` - Correlation matrix for a DataTable
- `skewness` - Skewness of a DataList
- `kurtosis` - Kurtosis of a DataList

## Transform / Time-Series
- `rank` - Rank DataList
- `normalize` - Normalize DataList
- `standardize` - Standardize DataList
- `reverse` - Reverse DataList
- `upper` - Uppercase DataList strings
- `lower` - Lowercase DataList strings
- `capitalize` - Capitalize DataList strings
- `parsenums` - Parse DataList strings to numbers
- `parsestrings` - Parse DataList numbers to strings
- `movavg` - Moving average
- `expsmooth` - Exponential smoothing
- `diff` - Difference (legacy, length n-1)
- `diffn` - Backward difference, same-length output with leading nils
- `shift` - Shift / lag / lead a DataList
- `pctchange` - Percent change over N rows
- `cumsum` - Running total
- `cumprod` - Running product
- `cummax` - Running maximum (historical high)
- `cummin` - Running minimum (historical low)
- `rolling` - Rolling-window reduction (sum/mean/min/max/median/std/var)
- `expanding` - Expanding-window reduction (sum/mean/min/max/median/std/var)
- `fillna` - Fill missing DataList/DataTable values (mean/median/mode/ffill/bfill/interpolate)
- `fillnan` - Fill NaN with mean (deprecated alias)

## Modeling / Inference / Visualization / Fetch
- `regression` - Regression analysis: linear/poly/exp/log/logistic/poisson
- `pca` - Principal component analysis
- `kmeans` - K-means clustering
- `hclust` - Hierarchical agglomerative clustering
- `cutree` - Cut a hierarchical clustering tree
- `dbscan` - Density-based clustering
- `silhouette` - Silhouette analysis
- `knn_classify` - K-nearest neighbors classification
- `knn_regress` - K-nearest neighbors regression
- `knn_neighbors` - K-nearest neighbors search
- `ttest` - T-test commands
- `ztest` - Z-test commands
- `anova` - ANOVA commands
- `ftest` - F-test commands
- `chisq` - Chi-square test commands
- `plot` - Create charts from variables
- `fetch` - Fetch external data

## Missing-Value Fill Commands

- `fillna <var> mean|median|mode|ffill|bfill|interpolate [cols A,B,C] [limit N] [extrapolate yes|no] [missing nan|nil|both] [as <var>]`
  - Works on either a DataList or a DataTable; saves a cloned result to `as <var>` or `$result`.
  - `cols` filters which DataTable columns to fill (omitted = all applicable); ignored for DataList input.
  - `limit` applies to `ffill` and `bfill`; `0` means unlimited.
  - `extrapolate yes` lets interpolation fill leading/trailing numeric gaps.
  - `missing` selects which kind of missing to fill: `nan`, `nil`, or `both` (default `both`).
  - `mean`, `median`, and `interpolate` skip non-numeric columns; `mode`, `ffill`, and `bfill` can fill any selected column type.
- `fillnan <var> mean [as <var>]`
  - Deprecated alias kept for backward compatibility. Only fills NaN (leaves nil alone) and only supports `mean`. Use `fillna <var> mean missing nan` instead.
