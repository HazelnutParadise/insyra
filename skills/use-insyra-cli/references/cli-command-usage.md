# Insyra CLI Command Usage (Full)

Generated from `insyra help` + `insyra help <command>` in current repository state.

For expanded subcommand forms and practical examples, see `cli-command-guide.md`.

## Conventions

### Literal value parsing

Commands that take a "value" argument (`addcol`, `set`, `shift ... fill ...`, `replace`, `pivot ... fillna ...`, `encode ordinal ... order ...`, `load sql ... params ...`, etc.) coerce each token through this ladder (case-insensitive for the keyword rows):

| Token                                  | Parsed as           |
| -------------------------------------- | ------------------- |
| `nil`                                  | Go `nil`            |
| `true` / `false`                       | bool                |
| `123` / `-7`                           | int                 |
| `1.5` / `1e3` / `.25` / `-2.5`         | float64             |
| `nan` / `NaN` / `NAN`                  | `math.NaN()`        |
| `inf` / `Inf` / `infinity` / `+inf`    | `math.Inf(+1)`      |
| `-inf` / `-Inf` / `-infinity`          | `math.Inf(-1)`      |
| anything else                          | string (as typed)   |

Because the float row dispatches through Go's `strconv.ParseFloat`, the tokens `nan`/`inf`/`infinity` are recognised as IEEE-754 special values, **not** as literal strings. If you genuinely need the string `"nan"` itself, pick a different token (e.g. `missing`).

This is separate from boolean-flag parsing used by option arguments like `headers true|false`, `center yes|no`, `rownames 1|0` — those accept only `yes/no/on/off/1/0/true/false`.

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

## `cummax`
- Description: Running maximum (historical high)
- Usage: `cummax <var> [as <var>]`

## `cummin`
- Description: Running minimum (historical low)
- Usage: `cummin <var> [as <var>]`

## `cumprod`
- Description: Running product
- Usage: `cumprod <var> [as <var>]`

## `cumsum`
- Description: Running total
- Usage: `cumsum <var> [as <var>]`

## `cutree`
- Description: Cut a hierarchical clustering tree
- Usage: `cutree <tree_var> k <n>|h <value> [as <var>]`

## `db`
- Description: Manage named database connections (sqlite, mysql, postgres; pure-Go drivers)
- Usage: `db connect <name> <dsn> | db list | db tables <name> [schema <s>] | db disconnect <name>`
- DSN forms:
	- `sqlite:<path-or-uri>` (e.g. `sqlite::memory:`, `sqlite:./foo.db`, `sqlite:file:./foo.db?mode=ro`)
	- `mysql:<go-sql-driver-dsn>` (e.g. `mysql:user:pass@tcp(host:3306)/db`)
	- `mysql://user:pass@host:port/db?param=value` (URL form, auto-converted)
	- `postgres://user:pass@host:port/db?sslmode=disable` (pgx URL form)
	- `postgres:host=... user=... password=... dbname=...` (libpq KV form)
- Notes:
	- Connection name must be unique within the environment.
	- `db list` masks passwords.
	- `db tables` defaults to current schema/database; pass `schema <s>` to override (mysql/postgres).
	- Connections do not persist across CLI process restarts; reopen at start of each session/script.

## `dbscan`
- Description: Density-based clustering
- Usage: `dbscan <var> <eps> <minpts> [as <var>]`

## `diff`
- Description: Difference (legacy, length n-1)
- Usage: `diff <var> [as <var>]`

## `diffn`
- Description: Backward difference, same-length output with leading nils
- Usage: `diffn <var> <periods> [as <var>]`

## `drop`
- Description: Delete variable
- Usage: `drop <var>`

## `dropcol`
- Description: Drop columns by name or index
- Usage: `dropcol <var> <name|index...>`

## `droprow`
- Description: Drop rows by index or name
- Usage: `droprow <var> <index|name...>`

## `encode`
- Description: One-shot categorical encoding for DataTable variables (encoder state is not persisted)
- Usage: `encode <var> onehot|label|ordinal ... [as <var>]`
- Full forms:
	- `encode <var> onehot <col1[,col2,...]> [dropfirst true|false] [keeporiginal true|false] [nan category|error|skip] [unknown ignore|error|new] [prefix <p>] [sep <s>] [sortcats true|false] [as <var>]`
	- `encode <var> label <col> [newcol <name>] [sortby firstseen|lex|freq] [nan category|error|skip] [unknown ignore|error|new] [keeporiginal true|false] [as <var>]`
	- `encode <var> ordinal <col> order <v1,v2,...> [newcol <name>] [unknown error|ignore] [nan category|error|skip] [keeporiginal true|false] [as <var>]`
- Notes:
  - Works on DataTable variables only.
  - CLI does one-shot fit+transform; it does not save encoder state for later commands.
  - `nan`: `category`, `error`, `skip`.
  - `unknown`: `ignore`, `error`, `new` for onehot/label; `ignore`, `error` for ordinal CLI.
  - `order` values are comma-separated and parsed through the literal-value ladder.

## `scale`
- Description: Fit a reusable feature scaler and transform/inverse tables with it (stateful)
- Usage: `scale fit std|minmax|robust|maxabs <scalerVar> <tableVar> [range <min> <max>] cols <c1,c2,...>` / `scale transform|inverse <scalerVar> <tableVar> as <outVar>`
- Full forms:
	- `scale fit std <scalerVar> <tableVar> cols <c1,c2,...>`
	- `scale fit minmax <scalerVar> <tableVar> range <min> <max> cols <c1,c2,...>`
	- `scale fit robust <scalerVar> <tableVar> cols <c1,c2,...>`
	- `scale fit maxabs <scalerVar> <tableVar> cols <c1,c2,...>`
	- `scale transform <scalerVar> <tableVar> as <outVar>`
	- `scale inverse <scalerVar> <tableVar> as <outVar>`
- Notes:
  - Works on DataTable variables only.
  - Stateful: `scale fit` stores a scaler variable; `transform`/`inverse` reuse it. Fit on train, transform test with the same parameters (no leakage).
  - Scaler variables are session-only and not saved to a named environment.
  - `minmax` defaults to `[0,1]` when `range` is omitted; `range` is only valid for `minmax`.
  - `nil`/`NaN` are preserved and excluded from fitting; non-fitted columns pass through unchanged.

## `env`
- Description: Environment management
- Usage: `env <create|list|open|clear|export|import|delete|rename|info> [args]`

## `exit`
- Description: Exit REPL
- Usage: `exit`

## `expanding`
- Description: Expanding-window reduction (in[0..=i]). Reducers: sum, mean, min, max, median, std, var.
- Usage: `expanding <var> <minobs> <reducer> [as <var>]`

## `expsmooth`
- Description: Exponential smoothing
- Usage: `expsmooth <var> <alpha> [as <var>]`

## `fetch`
- Description: Fetch external data
- Usage: `fetch yahoo <ticker> <method> [params...] [as <var>]`
- Supported methods:
	- `quote`, `info`, `history`, `dividends`, `splits`, `actions`, `options`, `calendar`, `fastinfo`
	- `news [count]` (default count = `10`)

## `fillna`
- Description: Fill missing DataList/DataTable values
- Usage: `fillna <var> mean|median|mode|ffill|bfill|interpolate [cols A,B,C] [limit N] [extrapolate yes|no] [missing nan|nil|both] [as <var>]`

## `fillnan`
- Description: Fill NaN with mean (deprecated alias)
- Usage: `fillnan <var> mean [as <var>]`

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

## `groupby`
- Description: Group a DataTable and aggregate columns
- Usage: `groupby <var> by <col1>[,<col2>...] agg <col>:<op>[:<alias>] [<col>:<op>[:<alias>] ...] [as <var>]`
- Notes:
  - Ops: `sum`, `mean` (alias `avg`), `median`, `min`, `max`, `count` (non-nil), `countall` (group size), `std`/`stdev`, `stdp`/`stdevp`, `var`, `varp`, `first`, `last`, `nunique`.
  - Bare token `count` is shorthand for `:countall:count`.
  - Aliases default to `<col>_<op>`.
  - Output column order: keys first (in `by` order), then aggregates (in `agg` order).
  - Group order: each unique key combination in first-seen order.

## `help`
- Description: Show command help
- Usage: `help [command]`

## `history`
- Description: Show command history
- Usage: `history`

## `iqr`
- Description: DataList IQR
- Usage: `iqr <var>`

## `kmeans`
- Description: K-means clustering
- Usage: `kmeans <var> <k> [nstart <n>] [itermax <n>] [seed <n>] [as <var>]`
- Side variables (auto-stored when result alias is `R`): `R_centers`, `R_size`, `R_withinss`, `R_totss`, `R_totwithinss`, `R_betweenss`, `R_iter`, `R_ifault`.

## `knn_classify`
- Description: K-nearest neighbors classification
- Usage: `knn_classify <train_var> <labels_var> <test_var> <k> [weighting <uniform|distance>] [algorithm <auto|brute|kd_tree|ball_tree>] [leafsize <n>] [as <var>]`
- Side variables: `<alias>_classes`, `<alias>_probs`.

## `knn_neighbors`
- Description: K-nearest neighbors search
- Usage: `knn_neighbors <train_var> <test_var> <k> [algorithm <auto|brute|kd_tree|ball_tree>] [leafsize <n>] [as <var>]`
- Side variables: `<alias>_distances`.

## `knn_regress`
- Description: K-nearest neighbors regression
- Usage: `knn_regress <train_var> <targets_var> <test_var> <k> [weighting <uniform|distance>] [algorithm <auto|brute|kd_tree|ball_tree>] [leafsize <n>] [as <var>]`

## `kurtosis`
- Description: Kurtosis of a DataList
- Usage: `kurtosis <var>`

## `hclust`
- Description: Hierarchical agglomerative clustering
- Usage: `hclust <var> <method> [as <var>]`

## `load`
- Description: Load data into a DataTable variable from a file, parquet, or SQL connection
- Usage: `load <file> [headers true|false] [rownames true|false] [encoding <enc>] [sheet <name>] | load parquet <file> [cols <c1,c2,...>] [rowgroups <i1,i2,...>] | load sql <conn> <table> [where "..."] [order "..."] [limit N] [offset N] [cols "c1,c2"] [schema <s>] [indexcol <c>] [parsedates "c1,c2"] | load sql <conn> query "<SQL>" [params <v1> <v2> ...] [as <var>]`
- File options (CSV / Excel):
	- `headers true|false` — first row is column names. Default `true`. JSON ignores this option (warns on use); Excel respects it.
	- `rownames true|false` — first column is row names. Default `false`.
	- `encoding <enc>` — CSV-only read-side hint (e.g. `big5`, `gbk`). Auto-detect when omitted.
	- `sheet <name>` — Excel-only; required for `.xlsx`/`.xlsm`/`.xls`.
	- Booleans accept `true|false|yes|no|on|off|1|0` (case-insensitive).
- SQL options:
	- Table form: `where "<expr>"`, `order "<expr>"`, `limit N`, `offset N`, `cols "c1,c2,..."`, `schema <s>`, `indexcol <c>`, `parsedates "c1,c2"`.
	- Query form: only `params <v1> <v2> ...` (positional bind values, parsed as literals).

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

## `pctchange`
- Description: Percent change over `periods` rows
- Usage: `pctchange <var> <periods> [as <var>]`

## `percentile`
- Description: DataList percentile
- Usage: `percentile <var> <p>`

## `pivot`
- Description: Reshape long-form DataTable to wide form (long -> wide)
- Usage: `pivot <var> index <col1[,col2,...]> columns <col> values <col> [agg <op>] [fillna <literal>] [sortcols true|false] [as <var>]`
- Notes:
  - `index` accepts comma-separated columns; `columns` and `values` take a single column.
  - Column tokens are resolved by `column.name` first, then fall back to Excel-style alphabetic index (`A`, `B`, ..., `AA`). The first row of data is never used as a header. Unknown tokens are an error.
  - Ops for `agg`: `sum`, `mean` (alias `avg`), `median`, `min`, `max`, `count` (non-nil), `countall` (group size), `std`/`stdev`, `stdp`/`stdevp`, `var`, `varp`, `first`, `last`, `nunique`.
  - When `agg` is omitted, duplicate `(index, columns)` combinations are an error.
  - `fillna <literal>` is parsed via the literal-value ladder (see below): nil → Go nil; true/false → bool; integer → int; float → float64 (incl. `nan` → math.NaN, `inf`/`infinity` → math.Inf(+1), `-inf` → math.Inf(-1), case-insensitive); else → string.
  - `sortcols true` orders generated columns by key value; default is first-seen.
  - Output column order: index first (in `index` order), then one column per unique `columns` value.

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
- Usage: `read <file> [headers true|false] [rownames true|false] [encoding <enc>] [sheet <name>]`
- Notes: forwards the file-side options to `load`; result is shown but not stored.

## `regression`
- Description: Regression analysis: linear/poly/exp/log/logistic/poisson
- Usage: `regression <type> <y> <x...>`
- Full forms:
	- `regression linear <y> <x1> [x2 ...] [as <var>]`
	- `regression poly <y> <x> <degree> [as <var>]`
	- `regression exp <y> <x> [as <var>]`
	- `regression log <y> <x> [as <var>]`
	- `regression logistic <y> <x1> [x2 ...] [as <var>]`
	- `regression poisson <y> <x1> [x2 ...] [as <var>]`
- Examples:
	- `insyra regression logistic y x1 x2 as fit`
	- `insyra regression poisson y x1 x2`

## `rename`
- Description: Rename variable
- Usage: `rename <var> <new>`

## `replace`
- Description: Replace values in DataTable/DataList
- Usage: `replace <var> <old|nan|nil> <new>`

## `reverse`
- Description: Reverse DataList
- Usage: `reverse <var> [as <var>]`

## `rolling`
- Description: Rolling-window reduction. Reducers: sum, mean, min, max, median, std, var. `minobs` defaults to window; `center yes` anchors at the central row (pandas-style).
- Usage: `rolling <var> <window> <reducer> [minobs <n>] [center yes|no] [as <var>]`

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
- Description: Randomly sample or shuffle a DataList/DataTable
- Usage: `sample <var> <n>|frac <frac>|shuffle [replace true|false] [seed N] [as <var>]`

## `split`
- Description: Split a DataTable into train/test tables
- Usage: `split <var> train <frac> [shuffle true|false] [seed N] as <trainVar> <testVar>`

## `save`
- Description: Save a DataTable variable to a file or SQL connection
- Usage: `save <var> <file> [headers true|false] [rownames true|false] [bom true|false] | save <var> sql <conn> <table> [if-exists fail|replace|append] [batch N] [schema <s>] [rownames]`
- File options (CSV):
	- `headers true|false` — write column names as the first row. Default `true`.
	- `rownames true|false` — write row names as the first column. Default `false`.
	- `bom true|false` — write a UTF-8 BOM (helps Excel for Windows open Chinese CSVs cleanly). Default `false`.
	- JSON: only `headers` applies (controls whether values use column names as keys); `rownames`/`bom` are rejected.
	- Parquet: file options are not supported (rejected).
	- Booleans accept `true|false|yes|no|on|off|1|0` (case-insensitive).
- SQL options:
	- `if-exists fail|replace|append` (default: `fail`)
	- `batch N` — INSERT batch size
	- `schema <s>` — target schema (mysql/postgres)
	- `rownames` — flag, write the DataTable row names as an extra column

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

## `shift`
- Description: Shift / lag / lead a DataList. Positive periods shift right (lag); negative shift left (lead). Empty slots default to nil; override with `fill <value>`.
- Usage: `shift <var> <periods> [fill <value>] [as <var>]`

## `show`
- Description: Display data with optional range (supports negative and _)
- Usage: `show <var> [N] [M]`

## `silhouette`
- Description: Silhouette analysis
- Usage: `silhouette <var> <labels_var> [as <var>]`
- Side variables: `<alias>_avg` (average silhouette width).

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

## `describe`
- Description: Create a reusable summary DataTable
- Usage: `describe <var> [by <col1>[,<col2>...]] [all true|false] [percentiles <p1,p2,...>] [as <var>]`

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

## `unpivot`
- Description: Reshape wide-form DataTable to long form (wide -> long)
- Usage: `unpivot <var> idvars <col1[,col2,...]> [valuevars <col1[,col2,...]>] [varname <name>] [valuename <name>] [dropna true|false] [as <var>]`
- Notes:
  - `idvars` is required and accepts comma-separated columns.
  - Column tokens (in `idvars` and `valuevars`) are resolved by `column.name` first, then fall back to Excel-style alphabetic index (`A`, `B`, ..., `AA`). The first row of data is never used as a header. Unknown tokens are an error.
  - `valuevars` defaults to all non-`idvars` columns when omitted.
  - `varname` defaults to `variable`; `valuename` defaults to `value`. They must differ.
  - `dropna true` skips output rows whose value is nil or NaN.
  - Output schema: idvars (in `idvars` order), then `varname`, then `valuename`.

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

