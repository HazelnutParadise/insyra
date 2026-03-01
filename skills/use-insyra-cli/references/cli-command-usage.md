# Insyra CLI Command Usage (Full)

Generated from `insyra help` + `insyra help <command>` in current repository state.

For expanded subcommand forms and practical examples, see `cli-command-guide.md`.

## `addcol`
- Description: Add one column to DataTable
- Usage: `addcol <var> <values...>`

## `addcolccl`
- Description: Add DataTable column using CCL
- Usage: `addcolccl <var> <name> <expr>`

## `addrow`
- Description: Add one row to DataTable
- Usage: `addrow <var> <values...>`

## `anova`
- Description: ANOVA commands
- Usage: `anova oneway|twoway|repeated ...`
- Full forms:
	- `anova oneway <group1> <group2> [group3...]`
	- `anova twoway <aLevels> <bLevels> <cell1> <cell2> ...`
	- `anova repeated <subject1> <subject2> ...`

## `capitalize`
- Description: Capitalize DataList strings
- Usage: `capitalize <var> [as <var>]`

## `ccl`
- Description: Execute CCL statements on DataTable
- Usage: `ccl <var> <expression>`

## `chisq`
- Description: Chi-square test commands
- Usage: `chisq gof|indep ...`
- Full forms:
	- `chisq gof <var> [p1 p2 ...]`
	- `chisq indep <rowVar> <colVar>`

## `clean`
- Description: Clean values from DataTable/DataList
- Usage: `clean <var> nan|nil|strings|outliers [<stddev>]`

## `clear`
- Description: Clear terminal screen
- Usage: `clear`

## `clone`
- Description: Deep clone DataTable/DataList variable
- Usage: `clone <var> [as <var>]`

## `col`
- Description: Extract DataTable column as DataList
- Usage: `col <var> <name|index> [as <var>]`

## `cols`
- Description: List DataTable column names
- Usage: `cols <var>`

## `config`
- Description: Read or update global CLI config
- Usage: `config [key] [value]`

## `convert`
- Description: Convert file formats (csv<->xlsx)
- Usage: `convert <input> <output>`

## `corr`
- Description: Correlation between two DataLists
- Usage: `corr <x> <y> [pearson|kendall|spearman]`

## `corrmatrix`
- Description: Correlation matrix for a DataTable
- Usage: `corrmatrix <datatable> [pearson|kendall|spearman] [as <var>]`

## `count`
- Description: Count occurrences
- Usage: `count <var> [value]`

## `counter`
- Description: DataList frequency map
- Usage: `counter <var>`

## `cov`
- Description: Covariance between two DataLists
- Usage: `cov <x> <y>`

## `diff`
- Description: Difference
- Usage: `diff <var> [as <var>]`

## `drop`
- Description: Delete variable
- Usage: `drop <var>`

## `dropcol`
- Description: Drop columns by name or index
- Usage: `dropcol <var> <name|index...>`

## `droprow`
- Description: Drop rows by index or name
- Usage: `droprow <var> <index|name...>`

## `env`
- Description: Environment management
- Usage: `env <create|list|open|clear|export|import|delete|rename|info> [args]`

## `exit`
- Description: Exit REPL
- Usage: `exit`

## `expsmooth`
- Description: Exponential smoothing
- Usage: `expsmooth <var> <alpha> [as <var>]`

## `fetch`
- Description: Fetch external data
- Usage: `fetch yahoo <ticker> <method> [params...] [as <var>]`
- Supported methods:
	- `quote`, `info`, `history`, `dividends`, `splits`, `actions`, `options`, `calendar`, `fastinfo`
	- `news [count]` (default count = `10`)

## `fillnan`
- Description: Fill NaN with mean
- Usage: `fillnan <var> mean`

## `filter`
- Description: Filter DataTable by CCL expression
- Usage: `filter <var> <expr> [as <var>]`

## `find`
- Description: Find rows containing value
- Usage: `find <var> <value>`

## `ftest`
- Description: F-test commands
- Usage: `ftest var|levene|bartlett ...`
- Full forms:
	- `ftest var <var1> <var2>`
	- `ftest levene <group1> <group2> [group3...]`
	- `ftest bartlett <group1> <group2> [group3...]`

## `get`
- Description: Get single element from DataTable
- Usage: `get <var> <row> <col>`

## `help`
- Description: Show command help
- Usage: `help [command]`

## `history`
- Description: Show command history
- Usage: `history`

## `iqr`
- Description: DataList IQR
- Usage: `iqr <var>`

## `kurtosis`
- Description: Kurtosis of a DataList
- Usage: `kurtosis <var>`

## `load`
- Description: Load data file into DataTable variable
- Usage: `load <file>|parquet <file> [cols <c1,c2,...>] [rowgroups <i1,i2,...>] [sheet <name>] [as <var>]`

## `lower`
- Description: Lowercase DataList strings
- Usage: `lower <var> [as <var>]`

## `max`
- Description: DataList maximum
- Usage: `max <var>`

## `mean`
- Description: DataList mean
- Usage: `mean <var>`

## `median`
- Description: DataList median
- Usage: `median <var>`

## `merge`
- Description: Merge two DataTables
- Usage: `merge <var1> <var2> <direction> <mode> [on <cols>] [as <var>]`

## `min`
- Description: DataList minimum
- Usage: `min <var>`

## `mode`
- Description: DataList mode
- Usage: `mode <var>`

## `movavg`
- Description: Moving average
- Usage: `movavg <var> <window> [as <var>]`

## `newdl`
- Description: Create DataList manually
- Usage: `newdl <values...> [as <var>]`

## `newdt`
- Description: Create DataTable from DataList variables
- Usage: `newdt <dl_vars...> [as <var>]`

## `normalize`
- Description: Normalize DataList
- Usage: `normalize <var> [as <var>]`

## `parsenums`
- Description: Parse DataList strings to numbers
- Usage: `parsenums <var> [as <var>]`

## `parsestrings`
- Description: Parse DataList numbers to strings
- Usage: `parsestrings <var> [as <var>]`

## `pca`
- Description: Principal component analysis
- Usage: `pca <var> <n>`

## `percentile`
- Description: DataList percentile
- Usage: `percentile <var> <p>`

## `plot`
- Description: Create charts from variables
- Usage: `plot <type> <var> [options...] [save <file>]`
- Supported types:
	- `line`, `bar`, `scatter`
- Save behavior:
	- default output: `<type>.html`
	- with `save <file>`: `.png` uses PNG export, other extensions use HTML export

## `quartile`
- Description: DataList quartile
- Usage: `quartile <var> <q>`

## `range`
- Description: DataList range
- Usage: `range <var>`

## `rank`
- Description: Rank DataList
- Usage: `rank <var> [asc|desc|true|false] [as <var>]`

## `read`
- Description: Quick preview a file without saving variable
- Usage: `read <file>`

## `regression`
- Description: Regression analysis: linear/poly/exp/log
- Usage: `regression <type> <y> <x...>`
- Full forms:
	- `regression linear <y> <x1> [x2 ...] [as <var>]`
	- `regression poly <y> <x> <degree> [as <var>]`
	- `regression exp <y> <x> [as <var>]`
	- `regression log <y> <x> [as <var>]`

## `rename`
- Description: Rename variable
- Usage: `rename <var> <new>`

## `replace`
- Description: Replace values in DataTable/DataList
- Usage: `replace <var> <old|nan|nil> <new>`

## `reverse`
- Description: Reverse DataList
- Usage: `reverse <var> [as <var>]`

## `row`
- Description: Extract DataTable row as DataList
- Usage: `row <var> <index|name> [as <var>]`

## `rows`
- Description: List DataTable row names
- Usage: `rows <var>`

## `run`
- Description: Run DSL script file
- Usage: `run <script.isr>`

## `sample`
- Description: Simple random sample from DataTable
- Usage: `sample <var> <n> [as <var>]`

## `save`
- Description: Save DataTable variable to file
- Usage: `save <var> <file>`

## `set`
- Description: Set single element in DataTable
- Usage: `set <var> <row> <col> <value>`

## `setcolnames`
- Description: Set DataTable column names
- Usage: `setcolnames <var> <names...>`

## `setrownames`
- Description: Set DataTable row names
- Usage: `setrownames <var> <names...>`

## `shape`
- Description: Show shape of DataTable/DataList
- Usage: `shape <var>`

## `show`
- Description: Display data with optional range (supports negative and _)
- Usage: `show <var> [N] [M]`

## `skewness`
- Description: Skewness of a DataList
- Usage: `skewness <var>`

## `sort`
- Description: Sort DataTable by one column
- Usage: `sort <var> <col> [asc|desc]`

## `standardize`
- Description: Standardize DataList
- Usage: `standardize <var> [as <var>]`

## `stdev`
- Description: DataList standard deviation
- Usage: `stdev <var>`

## `sum`
- Description: DataList sum
- Usage: `sum <var>`

## `summary`
- Description: Show summary statistics
- Usage: `summary <var>`

## `swap`
- Description: Swap DataTable columns or rows
- Usage: `swap <var> col|row <a> <b>`

## `transpose`
- Description: Transpose DataTable
- Usage: `transpose <var> [as <var>]`

## `ttest`
- Description: T-test commands
- Usage: `ttest single|two|paired ...`
- Full forms:
	- `ttest single <var> <mu>`
	- `ttest two <var1> <var2> [equal|unequal]`
	- `ttest paired <var1> <var2>`

## `types`
- Description: Show value types of DataTable/DataList
- Usage: `types <var>`

## `upper`
- Description: Uppercase DataList strings
- Usage: `upper <var> [as <var>]`

## `var`
- Description: DataList variance
- Usage: `var <var>`

## `vars`
- Description: List variables in current environment
- Usage: `vars`

## `version`
- Description: Show insyra version
- Usage: `version`

## `ztest`
- Description: Z-test commands
- Usage: `ztest single|two ...`
- Full forms:
	- `ztest single <var> <mu> <sigma> [two-sided|greater|less]`
	- `ztest two <var1> <var2> <sigma1> <sigma2> [two-sided|greater|less]`

