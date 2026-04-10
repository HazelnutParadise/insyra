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

    "github.com/HazelnutParadise/insyra/engine/dsl"
)

func main() {
    session, err := dsl.NewSession("default", nil)
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

## Environment Model

Insyra persists state per environment under:

```text
~/.insyra/envs/<name>/
```

Each environment contains:

- `state.json`: serialized variables (`DataTable`, `DataList`, and raw values).
- `history.txt`: command history.
- `config.json`: environment-local config payload.

Default behavior:

- On first run, `default` environment is auto-created.
- CLI commands restore env variables before execution.
- REPL saves history and state continuously.
- DSL session `Execute` saves state after successful command.

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

### C. Go `engine/dsl` session flow

```go
package main

import (
    "bytes"
    "fmt"

    "github.com/HazelnutParadise/insyra/engine/dsl"
)

func main() {
    var out bytes.Buffer

    session, err := dsl.NewSession("demo", &out)
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
- **DataTable Structure / Access**: `addcol`, `addrow`, `dropcol`, `droprow`, `swap`, `transpose`, `rows`, `cols`, `row`, `col`, `get`, `set`, `setrownames`, `setcolnames`
- **Data Processing**: `filter`, `sort`, `sample`, `find`, `replace`, `clean`, `merge`, `ccl`, `addcolccl`
- **DataList Stats**: `sum`, `mean`, `median`, `mode`, `stdev`, `var`, `min`, `max`, `range`, `quartile`, `iqr`, `percentile`, `count`, `counter`, `corr`, `cov`, `corrmatrix`, `skewness`, `kurtosis`
- **Time Series / Transforms**: `rank`, `normalize`, `standardize`, `reverse`, `upper`, `lower`, `capitalize`, `parsenums`, `parsestrings`, `movavg`, `expsmooth`, `diff`, `fillnan`
- **Modeling / Viz / Fetch**: `regression`, `pca`, `kmeans`, `hclust`, `cutree`, `dbscan`, `silhouette`, `ttest`, `ztest`, `anova`, `ftest`, `chisq`, `plot`, `fetch`

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
| `diff` | `diff <var> [as <var>]` | Difference |
| `drop` | `drop <var>` | Delete variable |
| `dropcol` | `dropcol <var> <name\|index...>` | Drop columns by name or index |
| `droprow` | `droprow <var> <index\|name...>` | Drop rows by index or name |
| `env` | `env <create\|list\|open\|clear\|export\|import\|delete\|rename\|info> [args]` | Environment management |
| `exit` | `exit` | Exit REPL |
| `expsmooth` | `expsmooth <var> <alpha> [as <var>]` | Exponential smoothing |
| `fetch` | `fetch yahoo <ticker> <method> [params...] [as <var>]` | Fetch external data |
| `fillnan` | `fillnan <var> mean` | Fill NaN with mean |
| `filter` | `filter <var> <expr> [as <var>]` | Filter DataTable by CCL expression |
| `find` | `find <var> <value>` | Find rows containing value |
| `ftest` | `ftest var\|levene\|bartlett ...` | F-test commands |
| `get` | `get <var> <row> <col>` | Get single element from DataTable |
| `help` | `help [command]` | Show command help |
| `history` | `history` | Show command history |
| `iqr` | `iqr <var>` | DataList IQR |
| `kmeans` | `kmeans <var> <k> [nstart <n>] [itermax <n>] [seed <n>] [as <var>]` | K-means clustering |
| `kurtosis` | `kurtosis <var>` | Kurtosis of a DataList |
| `load` | `load <file>\|parquet <file> [cols <c1,c2,...>] [rowgroups <i1,i2,...>] [sheet <name>] [as <var>]` | Load data file into DataTable variable |
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
| `parsestrings` | `parsestrings <var> [as <var>]` | Parse DataList numbers to strings |
| `pca` | `pca <var> <n>` | Principal component analysis |
| `hclust` | `hclust <var> <method> [as <var>]` | Hierarchical agglomerative clustering |
| `cutree` | `cutree <tree_var> k <n>\|h <value> [as <var>]` | Cut a hierarchical clustering tree |
| `dbscan` | `dbscan <var> <eps> <minpts> [as <var>]` | Density-based clustering |
| `silhouette` | `silhouette <var> <labels_var> [as <var>]` | Silhouette analysis |
| `percentile` | `percentile <var> <p>` | DataList percentile |
| `plot` | `plot <type> <var> [options...] [save <file>]` | Create charts from variables |
| `quartile` | `quartile <var> <q>` | DataList quartile |
| `range` | `range <var>` | DataList range |
| `rank` | `rank <var> [asc\|desc\|true\|false] [as <var>]` | Rank DataList |
| `read` | `read <file>` | Quick preview a file without saving variable |
| `regression` | `regression <type> <y> <x...>` | Regression analysis: linear/poly/exp/log |
| `rename` | `rename <var> <new>` | Rename variable |
| `replace` | `replace <var> <old\|nan\|nil> <new>` | Replace values in DataTable/DataList |
| `reverse` | `reverse <var> [as <var>]` | Reverse DataList |
| `row` | `row <var> <index\|name> [as <var>]` | Extract DataTable row as DataList |
| `rows` | `rows <var>` | List DataTable row names |
| `run` | `run <script.isr>` | Run DSL script file |
| `sample` | `sample <var> <n> [as <var>]` | Simple random sample from DataTable |
| `save` | `save <var> <file>` | Save DataTable variable to file |
| `set` | `set <var> <row> <col> <value>` | Set single element in DataTable |
| `setcolnames` | `setcolnames <var> <names...>` | Set DataTable column names |
| `setrownames` | `setrownames <var> <names...>` | Set DataTable row names |
| `shape` | `shape <var>` | Show shape of DataTable/DataList |
| `show` | `show <var> [N] [M]` | Display data with optional range (supports negative and _) |
| `skewness` | `skewness <var>` | Skewness of a DataList |
| `sort` | `sort <var> <col> [asc\|desc]` | Sort DataTable by one column |
| `standardize` | `standardize <var> [as <var>]` | Standardize DataList |
| `stdev` | `stdev <var>` | DataList standard deviation |
| `sum` | `sum <var>` | DataList sum |
| `summary` | `summary <var>` | Show summary statistics |
| `swap` | `swap <var> col\|row <a> <b>` | Swap DataTable columns or rows |
| `transpose` | `transpose <var> [as <var>]` | Transpose DataTable |
| `ttest` | `ttest single\|two\|paired ...` | T-test commands |
| `types` | `types <var>` | Show value types of DataTable/DataList |
| `upper` | `upper <var> [as <var>]` | Uppercase DataList strings |
| `var` | `var <var>` | DataList variance |
| `vars` | `vars` | List variables in current environment |
| `version` | `version` | Show insyra version |
| `ztest` | `ztest single\|two ...` | Z-test commands |

## Troubleshooting

- **Unknown command**: run `insyra help` to list commands, then `insyra help <command>` for usage.
- **Variable not found**: use `vars` to inspect current environment variables.
- **Variable type mismatch**: many commands require specific variable types (`DataTable` vs `DataList`).
- **Excel load fails**: `load <file.xlsx> sheet <sheet-name> [as <var>]` requires `sheet <name>`.
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

