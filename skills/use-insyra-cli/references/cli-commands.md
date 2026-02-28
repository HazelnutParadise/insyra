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
- `load` - Load data file into DataTable variable
- `read` - Quick preview a file without saving variable
- `save` - Save DataTable variable to file
- `convert` - Convert file formats (csv<->xlsx)

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
- `diff` - Difference
- `fillnan` - Fill NaN with mean

## Modeling / Inference / Visualization / Fetch
- `regression` - Regression analysis: linear/poly/exp/log
- `pca` - Principal component analysis
- `ttest` - T-test commands
- `ztest` - Z-test commands
- `anova` - ANOVA commands
- `ftest` - F-test commands
- `chisq` - Chi-square test commands
- `plot` - Create charts from variables
- `fetch` - Fetch external data
