# Insyra CLI Command Guide (By Topic + Examples)

Generated from current command registry (`insyra help`, `insyra help <command>`), then organized by topic.

## Core / Session
### `help`
- Description: Show command help
- Usage: `help [command]`
- Example: `insyra help`

### `version`
- Description: Show insyra version
- Usage: `version`
- Example: `insyra version`

### `exit`
- Description: Exit REPL
- Usage: `exit`
- Example: `insyra exit`

### `history`
- Description: Show command history
- Usage: `history`
- Example: `insyra history`

### `clear`
- Description: Clear terminal screen
- Usage: `clear`
- Example: `insyra clear`

### `config`
- Description: Read or update global CLI config
- Usage: `config [key] [value]`
- Example: `insyra config`

### `run`
- Description: Run DSL script file
- Usage: `run <script.isr>`
- Example: `insyra run <script.isr>`

## Environment
### `env`
- Description: Environment management
- Usage: `env <create|list|open|clear|export|import|delete|rename|info> [args]`
- Example: `insyra env <create|list|open|clear|export|import|delete|rename|info>`
- Subcommands: `create`, `list`, `open`, `clear`, `export`, `import`, `delete`, `rename`, `info`.

### `vars`
- Description: List variables in current environment
- Usage: `vars`
- Example: `insyra vars`

### `drop`
- Description: Delete variable
- Usage: `drop <var>`
- Example: `insyra drop x`

### `clone`
- Description: Deep clone DataTable/DataList variable
- Usage: `clone <var> [as <var>]`
- Example: `insyra clone x`

### `rename`
- Description: Rename variable
- Usage: `rename <var> <new>`
- Example: `insyra rename x newname`

### `shape`
- Description: Show shape of DataTable/DataList
- Usage: `shape <var>`
- Example: `insyra shape x`

### `types`
- Description: Show value types of DataTable/DataList
- Usage: `types <var>`
- Example: `insyra types x`

### `show`
- Description: Display data with optional range (supports negative and _)
- Usage: `show <var> [N] [M]`
- Example: `insyra show x`

### `summary`
- Description: Show summary statistics
- Usage: `summary <var>`
- Example: `insyra summary x`

## Data Creation / IO
### `newdl`
- Description: Create DataList manually
- Usage: `newdl <values...> [as <var>]`
- Example: `insyra newdl 1 2 3`

### `newdt`
- Description: Create DataTable from DataList variables
- Usage: `newdt <dl_vars...> [as <var>]`
- Example: `insyra newdt x y`

### `load`
- Description: Load data file into DataTable variable
- Usage: `load <file>|parquet <file> [cols <c1,c2,...>] [rowgroups <i1,i2,...>] [sheet <name>] [as <var>]`
- Example: `insyra load parquet data.parquet cols id,amount rowgroups 0,1 as t`

### `read`
- Description: Quick preview a file without saving variable
- Usage: `read <file>`
- Example: `insyra read data.csv`

### `save`
- Description: Save DataTable variable to file
- Usage: `save <var> <file>`
- Example: `insyra save x data.csv`

### `convert`
- Description: Convert file formats (csv<->xlsx)
- Usage: `convert <input> <output>`
- Example: `insyra convert input.csv output.xlsx`

## DataTable Structure & Access
### `addcol`
- Description: Add one column to DataTable
- Usage: `addcol <var> <values...>`
- Example: `insyra addcol x 1 2 3`

### `addrow`
- Description: Add one row to DataTable
- Usage: `addrow <var> <values...>`
- Example: `insyra addrow x 1 2 3`

### `dropcol`
- Description: Drop columns by name or index
- Usage: `dropcol <var> <name|index...>`
- Example: `insyra dropcol x 0`

### `droprow`
- Description: Drop rows by index or name
- Usage: `droprow <var> <index|name...>`
- Example: `insyra droprow x 0`

### `swap`
- Description: Swap DataTable columns or rows
- Usage: `swap <var> col|row <a> <b>`
- Example: `insyra swap x col 0 1`

### `transpose`
- Description: Transpose DataTable
- Usage: `transpose <var> [as <var>]`
- Example: `insyra transpose x`

### `rows`
- Description: List DataTable row names
- Usage: `rows <var>`
- Example: `insyra rows x`

### `cols`
- Description: List DataTable column names
- Usage: `cols <var>`
- Example: `insyra cols x`

### `row`
- Description: Extract DataTable row as DataList
- Usage: `row <var> <index|name> [as <var>]`
- Example: `insyra row x <index|name>`

### `col`
- Description: Extract DataTable column as DataList
- Usage: `col <var> <name|index> [as <var>]`
- Example: `insyra col x <name|index>`

### `get`
- Description: Get single element from DataTable
- Usage: `get <var> <row> <col>`
- Example: `insyra get x 0 0`

### `set`
- Description: Set single element in DataTable
- Usage: `set <var> <row> <col> <value>`
- Example: `insyra set x 0 0 <value>`

### `setrownames`
- Description: Set DataTable row names
- Usage: `setrownames <var> <names...>`
- Example: `insyra setrownames x a b c`

### `setcolnames`
- Description: Set DataTable column names
- Usage: `setcolnames <var> <names...>`
- Example: `insyra setcolnames x a b c`

## Data Processing
### `filter`
- Description: Filter DataTable by CCL expression
- Usage: `filter <var> <expr> [as <var>]`
- Example: `insyra filter x <expr>`

### `sort`
- Description: Sort DataTable by one column
- Usage: `sort <var> <col> [asc|desc]`
- Example: `insyra sort x 0`

### `sample`
- Description: Simple random sample from DataTable
- Usage: `sample <var> <n> [as <var>]`
- Example: `insyra sample x 3`

### `find`
- Description: Find rows containing value
- Usage: `find <var> <value>`
- Example: `insyra find x <value>`

### `replace`
- Description: Replace values in DataTable/DataList
- Usage: `replace <var> <old|nan|nil> <new>`
- Example: `insyra replace x <old|nan|nil> newname`

### `clean`
- Description: Clean values from DataTable/DataList
- Usage: `clean <var> nan|nil|strings|outliers [<stddev>]`
- Example: `insyra clean x nan|nil|strings|outliers`

### `merge`
- Description: Merge two DataTables
- Usage: `merge <var1> <var2> <direction> <mode> [on <cols>] [as <var>]`
- Example: `insyra merge x1 x2 inner strict`

### `ccl`
- Description: Execute CCL statements on DataTable
- Usage: `ccl <var> <expression>`
- Example: `insyra ccl x <expression>`

### `addcolccl`
- Description: Add DataTable column using CCL
- Usage: `addcolccl <var> <name> <expr>`
- Example: `insyra addcolccl x name <expr>`

## DataList Statistics
### `sum`
- Description: DataList sum
- Usage: `sum <var>`
- Example: `insyra sum x`

### `mean`
- Description: DataList mean
- Usage: `mean <var>`
- Example: `insyra mean x`

### `median`
- Description: DataList median
- Usage: `median <var>`
- Example: `insyra median x`

### `mode`
- Description: DataList mode
- Usage: `mode <var>`
- Example: `insyra mode x`

### `stdev`
- Description: DataList standard deviation
- Usage: `stdev <var>`
- Example: `insyra stdev x`

### `var`
- Description: DataList variance
- Usage: `var <var>`
- Example: `insyra var x`

### `min`
- Description: DataList minimum
- Usage: `min <var>`
- Example: `insyra min x`

### `max`
- Description: DataList maximum
- Usage: `max <var>`
- Example: `insyra max x`

### `range`
- Description: DataList range
- Usage: `range <var>`
- Example: `insyra range x`

### `quartile`
- Description: DataList quartile
- Usage: `quartile <var> <q>`
- Example: `insyra quartile x 1`

### `iqr`
- Description: DataList IQR
- Usage: `iqr <var>`
- Example: `insyra iqr x`

### `percentile`
- Description: DataList percentile
- Usage: `percentile <var> <p>`
- Example: `insyra percentile x 0.9`

### `count`
- Description: Count occurrences
- Usage: `count <var> [value]`
- Example: `insyra count x`

### `counter`
- Description: DataList frequency map
- Usage: `counter <var>`
- Example: `insyra counter x`

### `corr`
- Description: Correlation between two DataLists
- Usage: `corr <x> <y> [pearson|kendall|spearman]`
- Example: `insyra corr x y`

### `cov`
- Description: Covariance between two DataLists
- Usage: `cov <x> <y>`
- Example: `insyra cov x y`

### `corrmatrix`
- Description: Correlation matrix for a DataTable
- Usage: `corrmatrix <datatable> [pearson|kendall|spearman] [as <var>]`
- Example: `insyra corrmatrix <datatable>`

### `skewness`
- Description: Skewness of a DataList
- Usage: `skewness <var>`
- Example: `insyra skewness x`

### `kurtosis`
- Description: Kurtosis of a DataList
- Usage: `kurtosis <var>`
- Example: `insyra kurtosis x`

## Transform / Time-Series
### `rank`
- Description: Rank DataList
- Usage: `rank <var> [asc|desc|true|false] [as <var>]`
- Example: `insyra rank x desc as x_rank`

### `normalize`
- Description: Normalize DataList
- Usage: `normalize <var> [as <var>]`
- Example: `insyra normalize x`

### `standardize`
- Description: Standardize DataList
- Usage: `standardize <var> [as <var>]`
- Example: `insyra standardize x`

### `reverse`
- Description: Reverse DataList
- Usage: `reverse <var> [as <var>]`
- Example: `insyra reverse x`

### `upper`
- Description: Uppercase DataList strings
- Usage: `upper <var> [as <var>]`
- Example: `insyra upper x`

### `lower`
- Description: Lowercase DataList strings
- Usage: `lower <var> [as <var>]`
- Example: `insyra lower x`

### `capitalize`
- Description: Capitalize DataList strings
- Usage: `capitalize <var> [as <var>]`
- Example: `insyra capitalize x`

### `parsenums`
- Description: Parse DataList strings to numbers
- Usage: `parsenums <var> [as <var>]`
- Example: `insyra parsenums x`

### `parsestrings`
- Description: Parse DataList numbers to strings
- Usage: `parsestrings <var> [as <var>]`
- Example: `insyra parsestrings x`

### `movavg`
- Description: Moving average
- Usage: `movavg <var> <window> [as <var>]`
- Example: `insyra movavg x 3`

### `expsmooth`
- Description: Exponential smoothing
- Usage: `expsmooth <var> <alpha> [as <var>]`
- Example: `insyra expsmooth x 0.3`

### `diff`
- Description: Difference
- Usage: `diff <var> [as <var>]`
- Example: `insyra diff x`

### `fillnan`
- Description: Fill NaN with mean
- Usage: `fillnan <var> mean`
- Example: `insyra fillnan x mean`

## Modeling / Inference / Visualization / Fetch
### `regression`
- Description: Regression analysis: linear/poly/exp/log
- Usage: `regression <type> <y> <x...>`
- Example: `insyra regression line y x1 x2`
- Full forms: `regression linear <y> <x1> [x2 ...] [as <var>]` / `regression poly <y> <x> <degree> [as <var>]` / `regression exp <y> <x> [as <var>]` / `regression log <y> <x> [as <var>]`

### `pca`
- Description: Principal component analysis
- Usage: `pca <var> <n>`
- Example: `insyra pca x 3`

### `ttest`
- Description: T-test commands
- Usage: `ttest single|two|paired ...`
- Example: `insyra ttest single`
- Full forms: `ttest single <var> <mu>` / `ttest two <var1> <var2> [equal|unequal]` / `ttest paired <var1> <var2>`

### `ztest`
- Description: Z-test commands
- Usage: `ztest single|two ...`
- Example: `insyra ztest single`
- Full forms: `ztest single <var> <mu> <sigma> [two-sided|greater|less]` / `ztest two <var1> <var2> <sigma1> <sigma2> [two-sided|greater|less]`

### `anova`
- Description: ANOVA commands
- Usage: `anova oneway|twoway|repeated ...`
- Example: `insyra anova oneway`
- Full forms: `anova oneway <group1> <group2> [group3...]` / `anova twoway <aLevels> <bLevels> <cell1> <cell2> ...` / `anova repeated <subject1> <subject2> ...`

### `ftest`
- Description: F-test commands
- Usage: `ftest var|levene|bartlett ...`
- Example: `insyra ftest var`
- Full forms: `ftest var <var1> <var2>` / `ftest levene <group1> <group2> [group3...]` / `ftest bartlett <group1> <group2> [group3...]`

### `chisq`
- Description: Chi-square test commands
- Usage: `chisq gof|indep ...`
- Example: `insyra chisq gof`
- Full forms: `chisq gof <var> [p1 p2 ...]` / `chisq indep <rowVar> <colVar>`

### `plot`
- Description: Create charts from variables
- Usage: `plot <type> <var> [options...] [save <file>]`
- Example: `insyra plot line x`
- Types: `line`, `bar`, `scatter`; default output is `<type>.html` unless `save <file>` is specified.

### `fetch`
- Description: Fetch external data
- Usage: `fetch yahoo <ticker> <method> [params...] [as <var>]`
- Example: `insyra fetch yahoo AAPL quote`
- Yahoo methods: `quote`, `info`, `history`, `dividends`, `splits`, `actions`, `options`, `news [count]`, `calendar`, `fastinfo`.

