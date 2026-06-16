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

### `describe`
- Description: Create a reusable summary DataTable
- Usage: `describe <var> [by <col1[,col2,...]>] [all true|false] [percentiles <p1,p2,...>] [as <var>]`
- Example: `insyra describe sales all true as summary`
- Grouped example: `insyra describe sales by region percentiles 0.1,0.5,0.9 as region_summary`

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
- Description: Load data into a DataTable variable from a file, parquet, or SQL connection
- Usage: `load <file> [headers true|false] [rownames true|false] [encoding <enc>] [sheet <name>] | load parquet <file> [cols <c1,c2,...>] [rowgroups <i1,i2,...>] | load sql <conn> <table> [where "..."] [order "..."] [limit N] [offset N] [cols "c1,c2"] [schema <s>] [indexcol <c>] [parsedates "c1,c2"] | load sql <conn> query "<SQL>" [params <v1> <v2> ...] [as <var>]`
- Defaults: `headers=true`, `rownames=false`. Booleans accept `true|false|yes|no|on|off|1|0`.
- Examples:
  - `insyra load data.csv as t`
  - `insyra load matrix.csv headers false as t` (no header row)
  - `insyra load gdp.csv rownames true as t` (first column = row names)
  - `insyra load legacy.csv encoding big5 as t`
  - `insyra load report.xlsx sheet 2025 rownames true as t`
  - `insyra load parquet data.parquet cols id,amount rowgroups 0,1 as t`
  - `insyra load sql main customers as customers`
  - `insyra load sql main query "SELECT * FROM orders WHERE year = ?" params 2025 as orders`

### `read`
- Description: Quick preview a file without saving variable
- Usage: `read <file> [headers true|false] [rownames true|false] [encoding <enc>] [sheet <name>]`
- Example: `insyra read data.csv`
- Note: forwards the same file options as `load`.

### `save`
- Description: Save a DataTable variable to a file or SQL connection
- Usage: `save <var> <file> [headers true|false] [rownames true|false] [bom true|false] | save <var> sql <conn> <table> [if-exists fail|replace|append] [batch N] [schema <s>] [rownames]`
- Defaults: `headers=true`, `rownames=false`, `bom=false`. Booleans accept `true|false|yes|no|on|off|1|0`.
- Examples:
  - `insyra save x data.csv`
  - `insyra save matrix data.csv headers false`
  - `insyra save gdp out.csv rownames true`
  - `insyra save report data.csv bom true` (UTF-8 BOM for Windows Excel)
  - `insyra save report sql main report_table if-exists replace batch 1000`

### `convert`
- Description: Convert file formats (csv<->xlsx)
- Usage: `convert <input> <output>`
- Example: `insyra convert input.csv output.xlsx`

## Database (sqlite / mysql / postgres, pure-Go drivers)

### `db`
- Description: Manage named database connections (sqlite, mysql, postgres; pure-Go drivers)
- Usage: `db connect <name> <dsn> | db list | db tables <name> [schema <s>] | db disconnect <name>`
- Example: `insyra db connect main sqlite:./demo.db`
- DSN forms:
  - `sqlite:<path-or-uri>`, e.g. `sqlite::memory:`, `sqlite:./foo.db`
  - `mysql:<go-sql-driver-dsn>`, e.g. `mysql:user:pass@tcp(host:3306)/db`
  - `mysql://user:pass@host:port/db?param=value` (URL form, auto-converted)
  - `postgres://user:pass@host:port/db?sslmode=disable` (pgx URL form)
  - `postgres:host=... user=... password=... dbname=...` (libpq KV form)
- Notes: connections are environment-scoped and not persisted across CLI process restarts. Reopen them at the top of each session/script. `db list` masks passwords.

### `load sql`
- Description: Load a SQL table or query result into a DataTable
- Usage:
  - `load sql <conn> <table> [where "..."] [order "..."] [limit N] [offset N] [cols "c1,c2"] [schema <s>] [indexcol <c>] [parsedates "c1,c2"] [as <var>]`
  - `load sql <conn> query "<SQL>" [params <v1> <v2> ...] [as <var>]`
- Example: `insyra load sql main query "SELECT * FROM orders WHERE region = ?" params APAC as orders`

### `save sql`
- Description: Write a DataTable to a SQL table
- Usage: `save <var> sql <conn> <table> [if-exists fail|replace|append] [batch N] [schema <s>] [rownames]`
- Example: `insyra save report sql main report_table if-exists replace batch 1000`

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
- Description: Randomly sample or shuffle a DataList/DataTable
- Usage: `sample <var> <n>|frac <frac>|shuffle [replace true|false] [seed N] [as <var>]`
- Example: `insyra sample x frac 0.1 seed 42 as preview`

### `split`
- Description: Split a DataTable into train/test tables
- Usage: `split <var> train <frac> [shuffle true|false] [seed N] as <trainVar> <testVar>`
- Example: `insyra split x train 0.8 seed 42 as train test`

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

### `groupby`
- Description: Group a DataTable and aggregate columns (split-apply-combine)
- Usage: `groupby <var> by <col1>[,<col2>...] agg <col>:<op>[:<alias>] [<col>:<op>[:<alias>] ...] [as <var>]`
- Example: `insyra groupby sales by region agg revenue:sum:total_rev qty:mean as report`
- Multi-key with row-count shorthand: `insyra groupby sales by region,product agg revenue:sum count as report2`
- Ops: `sum`, `mean` (alias `avg`), `median`, `min`, `max`, `count` (non-nil), `countall` (group size), `std`/`stdev`, `stdp`/`stdevp`, `var`, `varp`, `first`, `last`, `nunique`. Aliases default to `<col>_<op>`.

### `pivot`
- Description: Reshape long-form DataTable to wide form (long -> wide)
- Usage: `pivot <var> index <col1[,col2,...]> columns <col> values <col> [agg <op>] [fillna <literal>] [sortcols true|false] [as <var>]`
- Example: `insyra pivot sales index region columns product values amount agg sum fillna 0 sortcols true as wide`
- Ops accepted by `agg` match `groupby`. When `agg` is omitted, duplicate `(index, columns)` combinations are an error.

### `unpivot`
- Description: Reshape wide-form DataTable to long form (wide -> long)
- Usage: `unpivot <var> idvars <col1[,col2,...]> [valuevars <col1[,col2,...]>] [varname <name>] [valuename <name>] [dropna true|false] [as <var>]`
- Example: `insyra unpivot survey idvars id valuevars Q1,Q2,Q3 varname question valuename score as long`
- `valuevars` defaults to all non-`idvars` columns. `varname` defaults to `variable`, `valuename` to `value`. `dropna true` skips nil/NaN values.

### `encode`
- Description: One-shot categorical encoding for DataTable variables (encoder state is not persisted)
- Usage: `encode <var> onehot|label|ordinal ... [as <var>]`
- Examples: `insyra encode sales onehot region,channel dropfirst true as x` / `insyra encode sales label segment newcol segment_id sortby freq keeporiginal true as labeled` / `insyra encode survey ordinal satisfaction order low,medium,high unknown error as ranked`
- Full forms:
  - `encode <var> onehot <col1[,col2,...]> [dropfirst true|false] [keeporiginal true|false] [nan category|error|skip] [unknown ignore|error|new] [prefix <p>] [sep <s>] [sortcats true|false] [as <var>]`
  - `encode <var> label <col> [newcol <name>] [sortby firstseen|lex|freq] [nan category|error|skip] [unknown ignore|error|new] [keeporiginal true|false] [as <var>]`
  - `encode <var> ordinal <col> order <v1,v2,...> [newcol <name>] [unknown error|ignore] [nan category|error|skip] [keeporiginal true|false] [as <var>]`
- Notes: one-shot only; use Go encoders for reusable train/test `Transform`. `order` values are parsed as literals.

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
- Description: Difference (legacy, length n-1)
- Usage: `diff <var> [as <var>]`
- Example: `insyra diff x`

### `diffn`
- Description: Backward difference with same-length output and leading nils. Prefer this over `diff` when you need column-aligned results.
- Usage: `diffn <var> <periods> [as <var>]`
- Example: `insyra diffn price 1 as d1`

### `shift`
- Description: Shift / lag / lead a DataList. Positive periods shift right (lag); negative shift left (lead). Empty slots default to nil; override with `fill <value>`.
- Usage: `shift <var> <periods> [fill <value>] [as <var>]`
- Example: `insyra shift price 1 as prev_price`

### `pctchange`
- Description: Percent change over `periods` rows. Divide-by-zero / non-numeric cells emit nil.
- Usage: `pctchange <var> <periods> [as <var>]`
- Example: `insyra pctchange price 1 as ret`

### `cumsum`
- Description: Running total. Nil / non-numeric cells emit nil at that position but the accumulator continues (pandas `skipna=True`).
- Usage: `cumsum <var> [as <var>]`
- Example: `insyra cumsum pnl as cum_pnl`

### `cumprod`
- Description: Running product. Same nil semantics as `cumsum`.
- Usage: `cumprod <var> [as <var>]`
- Example: `insyra cumprod growth as compounded`

### `cummax`
- Description: Running maximum (historical high). Same nil semantics as `cumsum`.
- Usage: `cummax <var> [as <var>]`
- Example: `insyra cummax price as hwm`

### `cummin`
- Description: Running minimum (historical low). Same nil semantics as `cumsum`.
- Usage: `cummin <var> [as <var>]`
- Example: `insyra cummin price as trough`

### `rolling`
- Description: Rolling-window reduction. Reducers: sum, mean, min, max, median, std, var. `minobs` defaults to window; `center yes` anchors at the central row (pandas-style).
- Usage: `rolling <var> <window> <reducer> [minobs <n>] [center yes|no] [as <var>]`
- Example: `insyra rolling price 7 mean minobs 1 as ma7_soft`

### `expanding`
- Description: Expanding-window reduction over `in[0..=i]`. Reducers: sum, mean, min, max, median, std, var. Emits nil until `minobs` valid observations are available.
- Usage: `expanding <var> <minobs> <reducer> [as <var>]`
- Example: `insyra expanding pnl 1 sum as cumulative_pnl`

### `fillna`
- Description: Fill missing DataList/DataTable values
- Usage: `fillna <var> mean|median|mode|ffill|bfill|interpolate [cols A,B,C] [limit N] [extrapolate yes|no] [missing nan|nil|both] [as <var>]`
- Examples: `insyra fillna price median as price_filled` / `insyra fillna price ffill limit 2 missing nan as price_ffill` / `insyra fillna sales median cols revenue,cost as cleaned`
- Notes: works on DataList or DataTable; `cols` filters DataTable columns and is ignored for DataList input; `limit` applies to `ffill`/`bfill` (`0` = unlimited); `extrapolate yes` lets interpolation fill leading/trailing gaps; `missing` selects NaN-only, nil-only, or both (default `both`); `mean`/`median`/`interpolate` skip non-numeric columns; `mode`/`ffill`/`bfill` work on any selected column type.

### `fillnan` (deprecated)
- Description: Fill NaN with mean — legacy alias
- Usage: `fillnan <var> mean [as <var>]`
- Example: `insyra fillnan price mean as price_filled`
- Notes: only fills NaN (leaves nil alone) and only supports `mean`. Use `fillna <var> mean missing nan` for the same behaviour with the new command.

## Modeling / Inference / Visualization / Fetch
### `regression`
- Description: Regression analysis: linear/poly/exp/log/logistic/poisson
- Usage: `regression <type> <y> <x...>`
- Examples: `insyra regression logistic y x1 x2 as fit` / `insyra regression poisson y x1 x2`
- Full forms: `regression linear <y> <x1> [x2 ...] [as <var>]` / `regression poly <y> <x> <degree> [as <var>]` / `regression exp <y> <x> [as <var>]` / `regression log <y> <x> [as <var>]` / `regression logistic <y> <x1> [x2 ...] [as <var>]` / `regression poisson <y> <x1> [x2 ...] [as <var>]`

### `pca`
- Description: Principal component analysis
- Usage: `pca <var> <n>`
- Example: `insyra pca x 3`

### `kmeans`
- Description: K-means clustering
- Usage: `kmeans <var> <k> [nstart <n>] [itermax <n>] [seed <n>] [as <var>]`
- Example: `insyra kmeans x 3 seed 42 as labels`
- Side variables (alias `R`): `R_centers`, `R_size`, `R_withinss`, `R_totss`, `R_totwithinss`, `R_betweenss`, `R_iter`, `R_ifault`.

### `hclust`
- Description: Hierarchical agglomerative clustering
- Usage: `hclust <var> <method> [as <var>]`
- Example: `insyra hclust x ward as tree`

### `cutree`
- Description: Cut a hierarchical clustering tree
- Usage: `cutree <tree_var> k <n>|h <value> [as <var>]`
- Example: `insyra cutree tree k 3 as labels`

### `dbscan`
- Description: Density-based clustering
- Usage: `dbscan <var> <eps> <minpts> [as <var>]`
- Example: `insyra dbscan x 0.5 5 as labels`
- Side variable: `<alias>_isseed`.

### `silhouette`
- Description: Silhouette analysis
- Usage: `silhouette <var> <labels_var> [as <var>]`
- Example: `insyra silhouette x labels as widths`
- Side variable: `<alias>_avg` (average silhouette width).

### `knn_classify`
- Description: K-nearest neighbors classification
- Usage: `knn_classify <train_var> <labels_var> <test_var> <k> [weighting <uniform|distance>] [algorithm <auto|brute|kd_tree|ball_tree>] [leafsize <n>] [as <var>]`
- Example: `insyra knn_classify train labels test 5 weighting distance as preds`
- Side variables: `<alias>_classes`, `<alias>_probs`.

### `knn_regress`
- Description: K-nearest neighbors regression
- Usage: `knn_regress <train_var> <targets_var> <test_var> <k> [weighting <uniform|distance>] [algorithm <auto|brute|kd_tree|ball_tree>] [leafsize <n>] [as <var>]`
- Example: `insyra knn_regress train targets test 5 as preds`

### `knn_neighbors`
- Description: K-nearest neighbors search
- Usage: `knn_neighbors <train_var> <test_var> <k> [algorithm <auto|brute|kd_tree|ball_tree>] [leafsize <n>] [as <var>]`
- Example: `insyra knn_neighbors train test 5 algorithm kd_tree as nn`
- Side variable: `<alias>_distances`.

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

