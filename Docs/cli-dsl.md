# CLI + DSL Guide

This document covers the Insyra command-line workflow end to end:

- CLI one-shot commands (`insyra <command> ...`)
- interactive REPL (`insyra`)
- `.isr` script execution (`insyra run <script.isr>`)
- Go programmatic DSL API (`engine/dsl`)

It is intended as a practical quickstart plus a complete command index.

## Table of Contents

- [Overview](#overview)
- [Installation](#installation)
- [Execution Modes](#execution-modes)
- [Global Flags](#global-flags)
- [Environment Model](#environment-model)
- [DSL Syntax Rules](#dsl-syntax-rules)
  - [Literal Values](#literal-values)
- [CLI Script Runner vs Go DSL API](#cli-script-runner-vs-go-dsl-api)
- [Quickstart Workflows](#quickstart-workflows)
- [Command Groups](#command-groups)
- [Full Command Index (Appendix)](#full-command-index-appendix)
- [Troubleshooting](#troubleshooting)

## Overview

Insyra uses one command system across CLI, REPL, scripts, and Go DSL sessions.

- **CLI mode** is best for reproducible one-shot automation in shell pipelines.
- **REPL mode** is best for interactive exploration with command history and completion.
- **Script mode** (`run`) executes line-by-line commands from a `.isr` text file.
- **Go DSL API** (`engine/dsl`) lets you execute the same command language inside Go code.

All modes share the same command registry (`cli/commands`), variable model (`map[string]any` in execution context), and environment persistence under `~/.insyra`.

## Installation

Install the CLI binary:

```bash
go install github.com/HazelnutParadise/insyra/cmd/insyra@latest
```

The executable is installed to `$GOBIN` (or `$GOPATH/bin` if `$GOBIN` is unset).

Windows note:

- Add `%USERPROFILE%\\go\\bin` (or your `%GOBIN%`) to `PATH` if `insyra` is not found.
- Restart the terminal after updating `PATH`.

## Execution Modes

### 1. One-shot CLI

```bash
insyra newdl 1 2 3 as x
insyra mean x
```

Use this mode for repeatable shell commands.

### 2. REPL

```bash
insyra
```

REPL prompt format:

```text
insyra [env-name] >
```

Empty lines and `# comment` lines are ignored.

### 3. Script Mode (`.isr`)

```bash
insyra run pipeline.isr
```

Each non-empty, non-comment line is tokenized and dispatched as a command.

### 4. Go DSL Session API

```go
package main

import (
    "fmt"

    "github.com/HazelnutParadise/insyra/cli/env"
    "github.com/HazelnutParadise/insyra/engine/dsl"
)

func main() {
    session, err := dsl.NewSession(env.Default(), "default", nil)
    if err != nil {
        panic(err)
    }

    if err := session.Execute("newdl 1 2 3 as x"); err != nil {
        panic(err)
    }
    if err := session.Execute("mean x"); err != nil {
        panic(err)
    }

    fmt.Println("vars:", len(session.Context().Vars))
}
```

`Execute` accepts the same DSL syntax as REPL and `.isr`.

#### Custom environment storage location

`NewSession` takes an `*env.Manager` that owns where environment files live.
Pass `env.Default()` for the standard `<UserHomeDir>/.insyra/envs/` layout,
or `env.NewManager(basePath, envsDirName)` for a custom root and/or a
custom per-environment subfolder name:

```go
// /workspace/.idensyra/envs/<name>/
mgr := env.NewManager("/workspace/.idensyra", "")

// /workspace/.idensyra/insights/<name>/  (Idensyra's layout)
mgr := env.NewManager("/workspace/.idensyra", "insights")

session, err := dsl.NewSession(mgr, "default", &out)
```

Both arguments accept "" to fall back to the defaults
(`<UserHomeDir>/.insyra` and `"envs"`).

Each session keeps its own Manager, so multiple sessions can coexist in
the same process with different roots and not interfere:

```go
mgrA := env.NewManager("/wsA", "")
mgrB := env.NewManager("/wsB", "")

sessionA, _ := dsl.NewSession(mgrA, "default", outA)
sessionB, _ := dsl.NewSession(mgrB, "default", outB)
// sessionA reads/writes /wsA/envs/...; sessionB reads/writes /wsB/envs/...
```

The Manager is also useful on its own when you only need the storage layer
(listing envs, exporting, reading history) without spinning up a session:

```go
mgr := env.NewManager("/workspace/.idensyra", "insights")
envs, _ := mgr.List()
mgr.Create("scratch")
mgr.Export("scratch", "/tmp/backup.json")
```

## Global Flags

Available on the root command:

- `--env <name>`: choose environment (default: `default`).
- `--no-color`: disable colored output.
- `--log-level <debug|info|warning|fatal>`: set runtime log level.

Examples:

```bash
insyra --env exp1 newdl 10 20 30 as x
insyra --no-color show x
insyra --log-level debug help
```

These flags work in front of every command, including the ones that take raw values such as `newdl -1 2 3`.

## Environment Model

Environment names may contain only letters, digits, `.`, `_` and `-`, must start with a letter or digit, and may not contain `..`. A name is joined onto the environments directory, so anything else is refused before any file is touched.

Insyra persists state per environment under:

```text
~/.insyra/envs/<name>/
```

Each environment contains:

- `state.json`: serialized variables (`DataTable`, `DataList`, and raw values).
- `history.txt`: command history. A `db connect` line is stored with its password masked (`user:***@…`, `password=***`), and the file is created private to the user (mode 0600).
- `config.json`: environment-local config payload.

Default behavior:

- On first run, `default` environment is auto-created.
- CLI commands restore env variables before execution.
- REPL saves history and state continuously.
- DSL session `Execute` saves state after successful command.
- Variables that hold `NaN` or ±Inf (for example a CSV loaded with blank cells) are saved and restored intact.
- `run <script.isr>` never opens the interactive REPL: an `env open` line inside a script only switches the environment for the rest of the script. A script that runs itself stops after 16 nested `run` levels.

Export/import:

```bash
insyra env export exp1 ./exp1.json
insyra env import ./exp1.json exp1-copy --force
```

`env import` into a non-empty target fails unless `--force` is provided.

## DSL Syntax Rules

`.isr` and REPL/DSL line parsing rules:

- Empty lines are ignored.
- Lines beginning with `#` are comments.
- Tokens are split by spaces/tabs.
- Single and double quotes are supported.
- Backslash escaping is supported.

Variable alias behavior:

- Most creating/transform commands accept `as <var>`.
- If `as <var>` is omitted on supported commands, result defaults to `$result`.

Examples:

```text
newdl 1 2 3 as x
newdl 4 5 6
# second command writes to $result
```

### Literal Values

Several commands accept **literal data values** as arguments — places where a single cell value is being supplied. The CLI coerces each token through this ladder (case-insensitive for the keyword rows):

| Token                                  | Parsed as                  |
| -------------------------------------- | -------------------------- |
| `nil`                                  | Go `nil`                   |
| `true` / `false`                       | `bool`                     |
| `123` / `-7`                           | `int`                      |
| `1.5` / `1e3` / `.25` / `-2.5`         | `float64`                  |
| `nan` / `NaN` / `NAN`                  | `math.NaN()`               |
| `inf` / `Inf` / `infinity` / `+inf`    | `math.Inf(+1)`             |
| `-inf` / `-Inf` / `-infinity`          | `math.Inf(-1)`             |
| anything else                          | `string` (as typed)        |

The float row is dispatched by Go's `strconv.ParseFloat`, which is why `nan`, `inf`, and `infinity` are recognised as IEEE-754 special values rather than literal strings. **If you need the string `"nan"` itself, pick a different token** (e.g. `missing`) — quoting at the shell level only strips quotes before the command sees the argument, it does not promote the token back to a string.

Commands whose arguments go through this ladder include (non-exhaustive):

| Where it applies                                         | Token position                    |
| -------------------------------------------------------- | --------------------------------- |
| `shift <var> <periods> fill <value> [as <var>]`          | `<value>`                         |
| `addcol <var> <values...>`                               | every `<value>`                   |
| `set <var> <row> <col> <value>`                          | `<value>`                         |
| `replace <var> <old\|nan\|nil> <new>`                    | `<new>`                           |
| `pivot ... fillna <literal>`                             | `<literal>`                       |
| `load sql <conn> query "<SQL>" params <v1> <v2> ...`     | every `<v>`                       |

This is intentionally separate from the boolean-flag parsing used by option arguments like `headers true|false`, `center yes|no`, `rownames 1|0` — those go through `parseFlexBool` and accept `yes/no/on/off/1/0/true/false` but **not** numeric or special-float tokens. See the option-parsing convention in each command's `help` output.

#### Negative numbers in one-shot mode

A token starting with `-` looks like a flag to the one-shot CLI's argument parser. `newdl`, `addrow`, and `addcol` turn that parsing off, so negative values need nothing special:

```bash
insyra newdl 0.01 -0.004 0.02 as r
```

The REPL and `.isr` scripts never parsed flags, so they are unaffected either way. Any *other* command that takes a value starting with `-` needs a bare `--` before the values, which stops flag parsing for the rest of the line:

```bash
insyra set dt 0 A -- -2.5
insyra shift x 1 fill -- -1
```

`help <command>` still works for the commands with flag parsing disabled; `insyra newdl --help` does not, because the command no longer reads flags.

## CLI Script Runner vs Go DSL API

`insyra run <script.isr>` and `Session.ExecuteFile(path)` are intentionally different on error handling.

### CLI `run`

- Executes line-by-line.
- On line error, prints `line N: <error>` and **continues**.
- Finishes with `script complete`.

### Go `Session.ExecuteFile`

- Executes line-by-line.
- On first error, returns `line N: <error>` and **stops**.
- Caller decides retry/fallback behavior.

Use `run` for tolerant batch execution and `ExecuteFile` for fail-fast programmatic control.

## Quickstart Workflows

### A. End-to-end CLI flow (load -> filter -> summary -> save)

```bash
insyra --env demo load sales.csv as t
insyra --env demo filter t "['amount'] > 1000" as t_high
insyra --env demo summary t_high
insyra --env demo save t_high high_sales.csv
```

### B. `.isr` pipeline

`pipeline.isr`:

```text
# Build high-value subset
load sales.csv as t
filter t "['amount'] > 1000" as t_high
summary t_high
save t_high high_sales.csv
```

Run:

```bash
insyra --env demo run pipeline.isr
```

### A1. Controlling headers and row names for CSV / Excel

`load` and `save` accept shared options for delimited/spreadsheet formats:

- `headers true|false` — whether the first row holds column names. Default `true`.
- `rownames true|false` — whether the first column holds row names. Default `false`.
- `encoding <enc>` — read-side hint for CSVs that aren't UTF-8 (e.g. `big5`, `gbk`). Auto-detected when omitted.
- `infer true|false` — read-side, CSV only. `infer false` keeps every cell as its original string (no type inference): leading zeros in stock IDs survive, amounts stay verbatim, empty cells stay empty strings. Default `true`.
- `ragged true|false` — read-side, CSV only. `ragged true` pads short rows with empty cells and keeps extra cells in automatically named columns. Padded cells count as empty for type inference (an otherwise-integer column becomes float with `NaN`); combine with `infer false` to keep cells verbatim. Default `false`, so uneven rows remain strict errors unless enabled.
- `trimspace true|false` — read-side, CSV only. `trimspace true` ignores leading whitespace before fields, including before quoted fields. Default `false`.
- `bom true|false` — save-side, write a UTF-8 BOM (helps Windows Excel open Chinese CSVs). Default `false`.

Boolean values accept `true|false`, `yes|no`, `on|off`, `1|0` (case-insensitive).

```text
# Pure data matrix (no header, no row labels)
load matrix.csv headers false as t

# Header + first-column labels (e.g. country names, dates)
load gdp.csv rownames true as t

# Big5 legacy CSV
load legacy.csv encoding big5 as t

# Stock IDs / exact amounts: load everything as raw strings
load stocks.csv infer false as raw

# Noisy CSV: trailer rows, extra cells, and whitespace before quotes
load inventory.csv ragged true trimspace true as inventory

# Excel: 'sheet' is required; headers/rownames are optional
load report.xlsx sheet 2025 rownames true as t

# Save with BOM so Excel for Windows opens it cleanly
save report data.csv bom true

# Save the row names back into the first column
save gdp out.csv rownames true

# Pure data dump, no header row
save matrix data.csv headers false
```

`read` is a quick previewer that forwards the same options to `load` (e.g. `read big5.csv encoding big5`).

### B0. SQL workflow (load and save against a live DB)

Open a named connection, list tables, load a query into a DataTable, transform, and write back:

```text
# pure-Go drivers; supports sqlite, mysql, postgres
db connect main sqlite:./demo.db
db list
db tables main

# load a whole table or a parameterized query
load sql main customers as customers
load sql main query "SELECT region, SUM(amount) AS total FROM orders WHERE year = ? GROUP BY region" params 2025 as totals

# transform with the usual commands, then write back
filter totals "['total'] > 10000" as top_regions
save top_regions sql main top_regions if-exists replace

db disconnect main
```

`load sql <conn> <table>` accepts `where "<sql>"`, `order "<sql>"`, `limit N`, `offset N`, `cols "c1,c2"`, `schema <s>`, `indexcol <c>`, and `parsedates "c1,c2"`. `load sql <conn> query "<SQL>"` accepts only `params <v1> <v2> ...` (positional placeholders, parsed as literals).

`save <var> sql <conn> <table>` accepts `if-exists fail|replace|append` (default `fail`), `batch N`, `schema <s>`, and the `rownames` flag (writes the DataTable row names as an extra column).

DSN forms accepted by `db connect`:

- `sqlite:<path-or-uri>` — e.g. `sqlite::memory:`, `sqlite:./foo.db`, `sqlite:file:./foo.db?mode=ro`
- `mysql:<go-sql-driver-dsn>` — e.g. `mysql:user:pass@tcp(host:3306)/db`
- `mysql://user:pass@host:port/db?param=value` (URL form, auto-converted)
- `postgres://user:pass@host:port/db?sslmode=disable` (URL form, native to pgx)
- `postgres:host=... user=... password=... dbname=...` (libpq KV form)

Passwords are masked when listed with `db list`.

### B2. GroupBy aggregations

`groupby <var> by <col1>[,<col2>...] agg <spec> [<spec> ...] [as <var>]` runs split-apply-combine. Each `<spec>` is `<col>:<op>[:<alias>]`, plus the shorthand `count` for "row count per group". Supported ops: `sum`, `mean` (alias `avg`), `median`, `min`, `max`, `count`, `countall`, `std` (`stdev`), `stdp` (`stdevp`), `var`, `varp`, `first`, `last`, `nunique`. Aliases default to `<col>_<op>`.

```text
load sales.csv as sales
groupby sales by region agg revenue:sum:total_rev qty:mean as report
show report

# Multi-key + count + named aggregate
groupby sales by region,product agg revenue:sum count as report2
```

The result is a fresh DataTable with one row per group; key columns appear first (in `by` order), aggregate columns next (in `agg` order).

### B3. Pivot / Unpivot (long ↔ wide reshape)

`pivot <var> index <col1[,col2,...]> columns <col> values <col> [agg <op>] [fillna <literal>] [sortcols true|false] [as <var>]` reshapes long-form data into wide form. The unique values of the `columns` column become new column headers; cells are filled from the `values` column; rows are keyed by the `index` columns.

```text
load sales.csv as sales
pivot sales index region columns product values amount agg sum fillna 0 sortcols true as wide
show wide
```

Supported `agg` ops match `groupby`: `sum`, `mean` (alias `avg`), `median`, `min`, `max`, `count`, `countall`, `std`/`stdev`, `stdp`/`stdevp`, `var`, `varp`, `first`, `last`, `nunique`. When `agg` is omitted and any `(index, columns)` pair has duplicates, the command errors.

`unpivot <var> idvars <col1[,col2,...]> [valuevars <col1[,col2,...]>] [varname <name>] [valuename <name>] [dropna true|false] [as <var>]` is the inverse: each input row is expanded into one output row per value column, with the source column name written to `varname` (default `variable`) and the cell value to `valuename` (default `value`). When `valuevars` is omitted it defaults to all non-`idvars` columns. `dropna true` skips rows whose value is nil or NaN.

```text
load survey.csv as survey
unpivot survey idvars id valuevars Q1,Q2,Q3 varname question valuename score as long
show long
```

### B4. Categorical encoding

`encode` does one-shot categorical encoding for DataTable variables: it fits on the input table, writes the encoded table to `as <var>` or `$result`, and does not persist encoder state across CLI commands. Use the Go API when you need train/test reuse with `Transform`.

```text
encode <var> onehot <col1[,col2,...]> [dropfirst true|false] [keeporiginal true|false] [nan category|error|skip] [unknown ignore|error|new] [prefix <p>] [sep <s>] [sortcats true|false] [as <var>]
encode <var> label <col> [newcol <name>] [sortby firstseen|lex|freq] [nan category|error|skip] [unknown ignore|error|new] [keeporiginal true|false] [as <var>]
encode <var> ordinal <col> order <v1,v2,...> [newcol <name>] [unknown error|ignore] [nan category|error|skip] [keeporiginal true|false] [as <var>]
```

Examples:

```text
encode sales onehot region,channel dropfirst true as x
encode sales label segment newcol segment_id sortby freq keeporiginal true as labeled
encode survey ordinal satisfaction order low,medium,high unknown error as ranked
```

### B5. Feature scaling

Unlike `encode`, `scale` is **stateful**: `scale fit` stores a reusable scaler variable, and `scale transform` / `scale inverse` apply that fitted scaler to any table. This lets you fit on a training set and transform a test set with the same parameters (no data leakage). Scaler variables live only for the session — they are not persisted to a named environment.

```text
scale fit std <scalerVar> <tableVar> cols <c1,c2,...>
scale fit minmax <scalerVar> <tableVar> range <min> <max> cols <c1,c2,...>
scale fit robust <scalerVar> <tableVar> cols <c1,c2,...>
scale fit maxabs <scalerVar> <tableVar> cols <c1,c2,...>
scale transform <scalerVar> <tableVar> as <outVar>
scale inverse <scalerVar> <tableVar> as <outVar>
```

`minmax` defaults to `[0,1]` when `range` is omitted. `nil`/`NaN` are preserved and ignored when fitting; non-fitted columns pass through unchanged. `show <scalerVar>` prints the scaler kind and its fitted columns.

Example (train/test without leakage):

```text
split dt train 0.8 as train test
scale fit std sc train cols Age,Income
scale transform sc train as train_scaled
scale transform sc test as test_scaled
scale inverse sc train_scaled as train_original
```

### B6. Exponential weighting and calendar resampling

`ewm <var> alpha|span|halflife <value> mean|var|std [adjust yes|no] [bias yes|no] [minobs <n>] [as <var>]` computes an exponentially weighted statistic over a DataList. Exactly one decay keyword is given: `alpha` in `(0, 1]`, `span >= 1` (`alpha = 2 / (span + 1)`), or `halflife > 0`. `adjust` and `bias` default to no, `minobs` to 1. The result is a DataList of the same length as the input.

```text
ewm price alpha 0.5 mean as ewma
ewm price span 12 mean adjust yes as ema12
ewm returns halflife 5 std minobs 3 as ewvol
```

`rolling` additionally takes two paired reducers, each consuming a second DataList variable: `rolling <var> <window> cov <other> [...]` and `rolling <var> <window> beta <other> [...]`. `beta` is `Cov(var, other) / Var(other)`; a flat benchmark window yields nil.

```text
rolling asset 20 cov benchmark as roll_cov
rolling asset 20 beta benchmark minobs 10 as roll_beta
```

`resample <dt> <timecol> weekly|monthly|quarterly|yearly <col>:<op>[:<name>] [...] [as <var>]` turns time-keyed rows into calendar-period aggregates. Each output row is labelled with the period's final calendar day, periods with no rows are omitted, and `op` accepts the same operator names as `groupby`. The optional third field renames the output column; without it the output keeps the source column name. Column names containing `:` cannot be written in this syntax.

```text
fetch yahoo AAPL history as bars
resample bars Date monthly Open:first High:max Low:min Close:last:MonthClose Volume:sum as monthly_bars
show monthly_bars
```

`<timecol>` must hold `time.Time` values; a column of date *strings* is rejected with a row-numbered error. A CSV load leaves date columns as strings, so convert the column with `parsedates` first (below), or start from a source that carries real timestamps such as `fetch yahoo <ticker> history`.

`parsedates <var> [cols <c1,c2>] [layout <go-layout>] [as <var>]` turns date strings into `time.Time`. On a DataList the whole list is converted; on a DataTable `cols` is required and names the columns to convert (by name, or by Excel index such as `A`). `layout` takes a Go reference layout and may be repeated — the layouts are tried in order and the first match wins. Without `layout`, common ISO shapes are tried (`2006-01-02`, `2006-01-02 15:04:05`, RFC 3339). A cell no layout matches becomes nil, so `resample` then reports it by row rather than converting half a column silently.

```text
load bars.csv as bars
parsedates bars cols Date as bars
resample bars Date monthly Close:last as monthly_close
```

```text
parsedates trades cols TradeDate,SettleDate layout 02/01/2006 as trades
```

### B7. Risk report from a return series

`quant` reaches the `quant` package: performance ratios, tail risk, market exposure, factor attribution, and European option pricing. Every series argument is a DataList variable of **per-period returns**, not prices, so convert a price column first — `pctchange` leaves a leading nil that `clean nil` removes.

```text
fetch yahoo AAPL history as bars
col bars Close as price
pctchange price 1 as ret
clean ret nil
quant sharpe ret 252 rf 0.0001 as sharpe
quant sortino ret 252 as sortino
quant var ret 0.95 as var95
quant cvar ret 0.95 parametric as cvar95
```

Each scalar form prints `name=value` and stores a `float64` under `as <var>` (or `$result`). `periods`, `days`, and `confidence` are required — the library refuses to guess an annualization factor, so there is no CLI-side default of 252. `rf`, `mar`, and `q` default to 0.

`capm` and `bs` store a one-row DataTable; `factor` stores one row per factor plus `<var>_alpha`; `drawdown` stores a DataList.

```text
quant capm asset market rf 0.0002 as capm
quant factor asset factors as fm
show fm
show fm_alpha
quant bs call 42 40 0.10 0.20 0.5 as opt
quant iv call 4.759 42 40 0.10 0.5
```

`portfolio` and `frontier` take a **DataTable** of aligned per-period returns — one column per asset, one row per period — instead of a single series:

```text
quant portfolio <returns_dt> minvar|target <r>|maxsharpe [rf <r>] [min <v1,...>] [max <v1,...>] [as <var>]
quant frontier <returns_dt> <points> [rf <r>] [min <v1,...>] [max <v1,...>] [as <var>]
```

`min` and `max` are comma-separated per-asset bounds in column order; the default box is long-only `[0, 1]`, so a short position needs an explicit negative `min`. A list whose length does not match the table's column count is refused before the solver runs. `rf` is per period and defaults to 0.

### B8. Taiwan stock beta from the CLI

`fetch tw` reaches the Taiwan Stock Exchange (TWSE) and Taipei Exchange (TPEx) daily datasets. No API key is needed, dates are written `YYYY-MM-DD`, and the trailing market keyword is `twse`, `tpex`, or `auto` (the default).

```text
fetch tw <code> prices <from> <to> [twse|tpex|auto] [as <var>]
fetch tw <code> adjprices <from> <to> [twse|auto] [as <var>]
fetch tw exrights <from> <to> [twse|auto] [as <var>]
fetch tw institutional <date> [twse|tpex|auto] [as <var>]
fetch tw margin <date> [twse|tpex|auto] [as <var>]
fetch tw quotes [twse|tpex|auto] [as <var>]
```

Use `adjprices` for anything that becomes a return series: on an ex-dividend or ex-rights day the quoted price drops without any loss to the holder, so `Close` shows a fake loss that `AdjClose` removes. `adjprices` and `exrights` are TWSE-only — TPEx publishes no dated ex-rights history, so passing `tpex` returns an explicit error rather than a silently unadjusted table.

Beta of TSMC against the 0050 market ETF, end to end:

```text
fetch tw 2330 adjprices 2026-01-01 2026-08-31 twse as tsmc
fetch tw 0050 adjprices 2026-01-01 2026-08-31 twse as market
col tsmc AdjClose as tsmc_px
col market AdjClose as market_px
pctchange tsmc_px 1 as tsmc_ret
pctchange market_px 1 as market_ret
clean tsmc_ret nil
clean market_ret nil
quant beta tsmc_ret market_ret as beta
```

Requests are throttled to one every 300 ms with two retries, which is what the exchanges' unauthenticated endpoints tolerate for a multi-year backfill. A script that needs a different pace sets the interval once:

```text
config fetch.tw.interval_ms 1000
```

The value is milliseconds and must be a non-negative integer; `0` turns throttling off.

### B9. Portfolio weights from the CLI

`quant portfolio` turns a table of aligned return columns into an allocation. Build the table by fetching each asset, taking returns, and putting the series side by side. `setcolnames` matters: `newdt` names each column after its DataList, and two series both extracted from an `AdjClose` column would otherwise arrive with the same name.

```text
fetch tw 2330 adjprices 2026-01-01 2026-08-31 twse as tsmc
fetch tw 2317 adjprices 2026-01-01 2026-08-31 twse as hon_hai
col tsmc AdjClose as tsmc_px
col hon_hai AdjClose as hon_hai_px
pctchange tsmc_px 1 as tsmc_ret
pctchange hon_hai_px 1 as hon_hai_ret
clean tsmc_ret nil
clean hon_hai_ret nil
newdt tsmc_ret hon_hai_ret as rets
setcolnames rets TSMC HonHai
quant portfolio rets maxsharpe rf 0.0001 as w
show w
show w_stats
```

`portfolio` prints one `<asset>=<weight>` line plus a summary (`return= vol= sharpe= iterations= converged=`). It stores a two-column `Asset, Weight` DataTable under `as <var>` (or `$result`), and a one-row `<var>_stats` table with `ExpectedReturn`, `Variance`, `Volatility`, `SharpeRatio`, `Iterations` and `Converged`. A solve that hits the iteration cap is reported as `converged=false` with the best weights found — it is not an error.

`frontier` sweeps the same problem across the attainable return range and stores one row per point: the fixed columns `ExpectedReturn`, `Variance`, `Volatility`, `SharpeRatio` and `Converged`, then one weight column per asset named after the asset. An asset literally named after one of those five columns is refused rather than silently renamed.

```text
quant frontier rets 20 min -0.2,-0.2 max 1,1 as f
save f frontier.csv
```

### C. Go `engine/dsl` session flow

```go
package main

import (
    "bytes"
    "fmt"

    "github.com/HazelnutParadise/insyra/cli/env"
    "github.com/HazelnutParadise/insyra/engine/dsl"
)

func main() {
    var out bytes.Buffer

    session, err := dsl.NewSession(env.Default(), "demo", &out)
    if err != nil {
        panic(err)
    }

    _ = session.Execute("load sales.csv as t")
    _ = session.Execute("filter t \"['amount'] > 1000\" as t_high")
    _ = session.Execute("summary t_high")
    _ = session.Execute("save t_high high_sales.csv")

    fmt.Println(out.String())
}
```

## Command Groups

High-level command map:

- **Core**: `help`, `version`, `exit`, `history`, `clear`, `config`, `run`, `completion`
- **Environment**: `env`, `vars`, `drop`, `clone`, `rename`, `shape`, `types`, `show`, `summary`
- **Data IO / Creation**: `newdl`, `newdt`, `load`, `read`, `save`, `convert`
- **Database**: `db` (`connect` / `list` / `tables` / `disconnect`), `load sql`, `save <var> sql`
- **DataTable Structure / Access**: `addcol`, `addrow`, `dropcol`, `droprow`, `swap`, `transpose`, `rows`, `cols`, `row`, `col`, `get`, `set`, `setrownames`, `setcolnames`
- **Data Processing**: `filter`, `sort`, `sample`, `split`, `find`, `replace`, `clean`, `fillna`, `merge`, `groupby`, `pivot`, `unpivot`, `encode`, `scale`, `ccl`, `addcolccl`
- **DataList Stats**: `sum`, `mean`, `median`, `mode`, `stdev`, `var`, `min`, `max`, `range`, `quartile`, `iqr`, `percentile`, `count`, `counter`, `corr`, `cov`, `corrmatrix`, `skewness`, `kurtosis`
- **Time Series / Transforms**: `rank`, `normalize`, `standardize`, `reverse`, `upper`, `lower`, `capitalize`, `parsenums`, `parsestrings`, `parsedates`, `movavg`, `expsmooth`, `diff`, `diffn`, `shift`, `pctchange`, `cumsum`, `cumprod`, `cummax`, `cummin`, `rolling`, `expanding`, `ewm`, `resample`, `fillna`
- **Modeling / Viz / Fetch**: `regression`, `pca`, `kmeans`, `hclust`, `cutree`, `dbscan`, `silhouette`, `knn_classify`, `knn_regress`, `knn_neighbors`, `ttest`, `ztest`, `anova`, `ftest`, `chisq`, `plot`, `fetch`
- **Quant**: `quant` (`sharpe`, `sortino`, `ir`, `maxdd`, `annret`, `calmar`, `drawdown`, `var`, `cvar`, `beta`, `capm`, `factor`, `bs`, `iv`, `portfolio`, `frontier`)

### Missing-Value Fill Commands

```bash
fillna <var> mean|median|mode|ffill|bfill|interpolate [cols A,B,C] [limit N] [extrapolate yes|no] [missing nan|nil|both] [as <var>]
fillnan <var> mean [as <var>]   # deprecated; only fills NaN, mean only
```

`fillna` clones the input (DataList or DataTable) and saves the filled result under `as <var>` or `$result`. `cols` filters which DataTable columns to touch (ignored for DataList). `limit` caps consecutive forward/backward fills; `extrapolate` controls whether interpolation fills leading/trailing gaps; `missing` selects which kind of missing to fill (default `both`). `mean`, `median`, and `interpolate` skip non-numeric columns; `mode`, `ffill`, and `bfill` work with any selected column type.

`fillnan <var> mean` is a legacy alias kept for backward compatibility — it only fills NaN (leaves nil alone) and only supports the `mean` strategy. New code should use `fillna ... missing nan` instead.

Examples:

```bash
fillna price interpolate extrapolate yes as price_interp
fillna status ffill limit 2 missing nan as status_filled
fillna sales median cols revenue,cost as sales_clean
fillna sales bfill limit 1 as sales_bfill
```

## Full Command Index (Appendix)

Source policy:

- Registry commands: `go run ./cmd/insyra help` and `go run ./cmd/insyra help <command>`.
- Cobra built-in command noted explicitly: `completion`.

| Command | Usage | Description |
| --- | --- | --- |
| `addcol` | `addcol <var> <values...>` | Add one column to DataTable |
| `addcolccl` | `addcolccl <var> <name> <expr>` | Add DataTable column using CCL |
| `addrow` | `addrow <var> <values...>` | Add one row to DataTable |
| `anova` | `anova oneway\|twoway\|repeated ...` | ANOVA commands |
| `capitalize` | `capitalize <var> [as <var>]` | Capitalize DataList strings |
| `ccl` | `ccl <var> <expression>` | Execute CCL statements on DataTable |
| `chisq` | `chisq gof\|indep ...` | Chi-square test commands |
| `clean` | `clean <var> nan\|nil\|strings\|outliers [<stddev>]` | Clean values from DataTable/DataList |
| `clear` | `clear` | Clear terminal screen |
| `clone` | `clone <var> [as <var>]` | Deep clone DataTable/DataList variable |
| `col` | `col <var> <name\|index> [as <var>]` | Extract DataTable column as DataList |
| `cols` | `cols <var>` | List DataTable column names |
| `completion` | `completion [command]` | Generate the autocompletion script for insyra for the specified shell. |
| `config` | `config [key] [value]` | Read or update global CLI config |
| `convert` | `convert <input> <output>` | Convert file formats (csv<->xlsx) |
| `corr` | `corr <x> <y> [pearson\|kendall\|spearman]` | Correlation between two DataLists |
| `corrmatrix` | `corrmatrix <datatable> [pearson\|kendall\|spearman] [as <var>]` | Correlation matrix for a DataTable |
| `count` | `count <var> [value]` | Count occurrences |
| `counter` | `counter <var>` | DataList frequency map |
| `cov` | `cov <x> <y>` | Covariance between two DataLists |
| `db` | `db connect <name> <dsn> \| db list \| db tables <name> [schema <s>] \| db disconnect <name>` | Manage named database connections (sqlite, mysql, postgres; pure-Go drivers) |
| `cummax` | `cummax <var> [as <var>]` | Running maximum (historical high) |
| `cummin` | `cummin <var> [as <var>]` | Running minimum (historical low) |
| `cumprod` | `cumprod <var> [as <var>]` | Running product |
| `cumsum` | `cumsum <var> [as <var>]` | Running total |
| `describe` | `describe <var> [by <col1[,col2,...]>] [all true\|false] [percentiles <p1,p2,...>] [as <var>]` | Create a programmatic summary table |
| `diff` | `diff <var> [as <var>]` | Difference (legacy, length n-1) |
| `diffn` | `diffn <var> <periods> [as <var>]` | Backward difference, same-length output with leading nils |
| `drop` | `drop <var>` | Delete variable |
| `dropcol` | `dropcol <var> <name\|index...>` | Drop columns by name or index |
| `droprow` | `droprow <var> <index\|name...>` | Drop rows by index or name |
| `encode` | `encode <var> onehot\|label\|ordinal ... [as <var>]` | One-shot categorical encoding for DataTable variables |
| `env` | `env <create\|list\|open\|clear\|export\|import\|delete\|rename\|info> [args]` | Environment management |
| `ewm` | `ewm <var> alpha\|span\|halflife <value> mean\|var\|std [adjust yes\|no] [bias yes\|no] [minobs <n>] [as <var>]` | Exponentially weighted mean/var/std over a DataList |
| `exit` | `exit` | Exit REPL |
| `expanding` | `expanding <var> <minobs> <reducer> [as <var>]` | Expanding-window reduction (reducer: sum\|mean\|min\|max\|median\|std\|var) |
| `expsmooth` | `expsmooth <var> <alpha> [as <var>]` | Exponential smoothing |
| `fetch` | `fetch yahoo <ticker> <method> [params...] [as <var>]` / `fetch tw [<code>] prices\|adjprices\|exrights\|institutional\|margin\|quotes ... [as <var>]` | Fetch external data |
| `fillna` | `fillna <var> mean\|median\|mode\|ffill\|bfill\|interpolate [cols A,B,C] [limit N] [extrapolate yes\|no] [missing nan\|nil\|both] [as <var>]` | Fill missing DataList/DataTable values |
| `fillnan` | `fillnan <var> mean [as <var>]` | Fill NaN with mean (deprecated alias) |
| `filter` | `filter <var> <expr> [as <var>]` | Filter DataTable by CCL expression |
| `find` | `find <var> <value>` | Find rows containing value |
| `ftest` | `ftest var\|levene\|bartlett ...` | F-test commands |
| `get` | `get <var> <row> <col>` | Get single element from DataTable |
| `groupby` | `groupby <var> by <col1>[,<col2>...] agg <col>:<op>[:<alias>] [<col>:<op>[:<alias>] ...] [as <var>]` | Group a DataTable and aggregate columns (split-apply-combine) |
| `help` | `help [command]` | Show command help |
| `history` | `history` | Show command history |
| `iqr` | `iqr <var>` | DataList IQR |
| `knn_classify` | `knn_classify <train_var> <labels_var> <test_var> <k> [weighting <uniform\|distance>] [algorithm <auto\|brute\|kd_tree\|ball_tree>] [leafsize <n>] [as <var>]` | K-nearest neighbors classification |
| `knn_regress` | `knn_regress <train_var> <targets_var> <test_var> <k> [weighting <uniform\|distance>] [algorithm <auto\|brute\|kd_tree\|ball_tree>] [leafsize <n>] [as <var>]` | K-nearest neighbors regression |
| `knn_neighbors` | `knn_neighbors <train_var> <test_var> <k> [algorithm <auto\|brute\|kd_tree\|ball_tree>] [leafsize <n>] [as <var>]` | K-nearest neighbors search |
| `kmeans` | `kmeans <var> <k> [nstart <n>] [itermax <n>] [seed <n>] [as <var>]` | K-means clustering |
| `kurtosis` | `kurtosis <var>` | Kurtosis of a DataList |
| `load` | `load <file> [headers true\|false] [rownames true\|false] [encoding <enc>] [infer true\|false] [sheet <name>] \| load parquet <file> [cols <c1,c2,...>] [rowgroups <i1,i2,...>] \| load sql <conn> <table> [where "..."] [order "..."] [limit N] [offset N] [cols "c1,c2"] [schema <s>] [indexcol <c>] [parsedates "c1,c2"] \| load sql <conn> query "<SQL>" [params <v1> <v2> ...] [as <var>]` | Load data into a DataTable variable from a file, parquet, or SQL connection |
| `lower` | `lower <var> [as <var>]` | Lowercase DataList strings |
| `max` | `max <var>` | DataList maximum |
| `mean` | `mean <var>` | DataList mean |
| `median` | `median <var>` | DataList median |
| `merge` | `merge <var1> <var2> <direction> <mode> [on <cols>] [as <var>]` | Merge two DataTables |
| `min` | `min <var>` | DataList minimum |
| `mode` | `mode <var>` | DataList mode |
| `movavg` | `movavg <var> <window> [as <var>]` | Moving average |
| `newdl` | `newdl <values...> [as <var>]` | Create DataList manually |
| `newdt` | `newdt <dl_vars...> [as <var>]` | Create DataTable from DataList variables |
| `normalize` | `normalize <var> [as <var>]` | Normalize DataList |
| `parsenums` | `parsenums <var> [as <var>]` | Parse DataList strings to numbers |
| `parsedates` | `parsedates <var> [cols <c1,c2>] [layout <go-layout>] [as <var>]` | Convert date strings to `time.Time` in a DataList or DataTable columns |
| `parsestrings` | `parsestrings <var> [as <var>]` | Parse DataList numbers to strings |
| `pca` | `pca <var> <n>` | Principal component analysis |
| `pctchange` | `pctchange <var> <periods> [as <var>]` | Percent change over `periods` rows |
| `pivot` | `pivot <var> index <col1[,col2,...]> columns <col> values <col> [agg <op>] [fillna <literal>] [sortcols true\|false] [as <var>]` | Reshape long-form DataTable to wide form |
| `hclust` | `hclust <var> <method> [as <var>]` | Hierarchical agglomerative clustering |
| `cutree` | `cutree <tree_var> k <n>\|h <value> [as <var>]` | Cut a hierarchical clustering tree |
| `dbscan` | `dbscan <var> <eps> <minpts> [as <var>]` | Density-based clustering |
| `silhouette` | `silhouette <var> <labels_var> [as <var>]` | Silhouette analysis |
| `percentile` | `percentile <var> <p>` | DataList percentile |
| `plot` | `plot <type> <var> [options...] [save <file>]` | Create charts from variables |
| `quant` | `quant sharpe\|sortino\|ir\|maxdd\|annret\|calmar\|drawdown\|var\|cvar\|beta\|capm\|factor\|bs\|iv\|portfolio\|frontier ...` | Quantitative finance: performance, risk, exposure, factor, option and portfolio analytics |
| `quartile` | `quartile <var> <q>` | DataList quartile |
| `range` | `range <var>` | DataList range |
| `rank` | `rank <var> [asc\|desc\|true\|false] [as <var>]` | Rank DataList |
| `read` | `read <file> [headers true\|false] [rownames true\|false] [encoding <enc>] [infer true\|false] [sheet <name>]` | Quick preview a file without saving variable |
| `regression` | `regression <type> <y> <x...>` | Regression analysis: linear/poly/exp/log/logistic/poisson |
| `rename` | `rename <var> <new>` | Rename variable |
| `replace` | `replace <var> <old\|nan\|nil> <new>` | Replace values in DataTable/DataList |
| `resample` | `resample <dt> <timecol> weekly\|monthly\|quarterly\|yearly <col>:<op>[:<name>] [...] [as <var>]` | Aggregate a time-indexed DataTable into calendar periods |
| `reverse` | `reverse <var> [as <var>]` | Reverse DataList |
| `rolling` | `rolling <var> <window> <reducer> [minobs <n>] [center yes\|no] [as <var>]` | Rolling-window reduction (reducer: sum\|mean\|min\|max\|median\|std\|var, or cov\|beta with a second DataList) |
| `row` | `row <var> <index\|name> [as <var>]` | Extract DataTable row as DataList |
| `rows` | `rows <var>` | List DataTable row names |
| `run` | `run <script.isr>` | Run DSL script file |
| `sample` | `sample <var> <n>\|frac <frac>\|shuffle [replace true\|false] [seed N] [as <var>]` | Randomly sample or shuffle a DataList/DataTable |
| `save` | `save <var> <file> [headers true\|false] [rownames true\|false] [bom true\|false] \| save <var> sql <conn> <table> [if-exists fail\|replace\|append] [batch N] [schema <s>] [rownames]` | Save a DataTable variable to a file or SQL connection |
| `scale` | `scale fit std\|minmax\|robust\|maxabs <scalerVar> <tableVar> [range <min> <max>] cols <c1,c2,...> \| scale transform\|inverse <scalerVar> <tableVar> as <outVar>` | Fit a reusable feature scaler and transform/inverse tables with it |
| `set` | `set <var> <row> <col> <value>` | Set single element in DataTable |
| `setcolnames` | `setcolnames <var> <names...>` | Set DataTable column names |
| `setrownames` | `setrownames <var> <names...>` | Set DataTable row names |
| `shape` | `shape <var>` | Show shape of DataTable/DataList |
| `shift` | `shift <var> <periods> [fill <value>] [as <var>]` | Shift / lag (periods > 0) / lead (periods < 0); empty slots default to nil |
| `show` | `show <var> [N] [M]` | Display data with optional range (supports negative and _) |
| `skewness` | `skewness <var>` | Skewness of a DataList |
| `sort` | `sort <var> <col> [asc\|desc]` | Sort DataTable by one column |
| `split` | `split <var> train <frac> [shuffle true\|false] [seed N] as <trainVar> <testVar>` | Split DataTable rows into train/test tables |
| `standardize` | `standardize <var> [as <var>]` | Standardize DataList |
| `stdev` | `stdev <var>` | DataList standard deviation |
| `sum` | `sum <var>` | DataList sum |
| `summary` | `summary <var>` | Show summary statistics |
| `swap` | `swap <var> col\|row <a> <b>` | Swap DataTable columns or rows |
| `transpose` | `transpose <var> [as <var>]` | Transpose DataTable |
| `ttest` | `ttest single\|two\|paired ...` | T-test commands |
| `types` | `types <var>` | Show value types of DataTable/DataList |
| `unpivot` | `unpivot <var> idvars <col1[,col2,...]> [valuevars <col1[,col2,...]>] [varname <name>] [valuename <name>] [dropna true\|false] [as <var>]` | Reshape wide-form DataTable to long form |
| `upper` | `upper <var> [as <var>]` | Uppercase DataList strings |
| `var` | `var <var>` | DataList variance |
| `vars` | `vars` | List variables in current environment |
| `version` | `version` | Show insyra version |
| `ztest` | `ztest single\|two ...` | Z-test commands |

## Describe Command

`describe` creates a DataTable summary that can be saved or reused in later commands.

```bash
describe sales as summary
describe sales all true as summary_all
describe sales percentiles 0.1,0.5,0.9 as summary_p
describe sales by region all true as region_summary
save region_summary region_summary.csv
```

Without `as`, the result is stored in `$result`. `all true` includes non-numeric and mixed columns. `by` is available for DataTable variables only.

## Regression Forms

The `regression` command supports:

- `regression linear <y> <x1> [x2 ...] [as <var>]`
- `regression poly <y> <x> <degree> [as <var>]`
- `regression exp <y> <x> [as <var>]`
- `regression log <y> <x> [as <var>]`
- `regression logistic <y> <x1> [x2 ...] [as <var>]`
- `regression poisson <y> <x1> [x2 ...] [as <var>]`

Examples:

```bash
insyra regression logistic y x1 x2 as fit
insyra regression poisson y x1 x2
```

## Troubleshooting

- **Unknown command**: run `insyra help` to list commands, then `insyra help <command>` for usage.
- **Variable not found**: use `vars` to inspect current environment variables.
- **Variable type mismatch**: many commands require specific variable types (`DataTable` vs `DataList`).
- **Excel load fails**: `load <file.xlsx> sheet <sheet-name> [headers true|false] [rownames true|false] [as <var>]` always requires `sheet <name>`.
- **Parquet option errors**:
  - `cols` and `rowgroups` must be followed by comma-separated values.
  - `rowgroups` must be non-negative integers.
  - unknown options are rejected.

Examples:

```bash
# valid
load parquet data.parquet cols id,amount,status rowgroups 0,1 as t

# invalid (unknown option)
load parquet data.parquet columns id
```
