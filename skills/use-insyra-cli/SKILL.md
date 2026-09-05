---
name: use-insyra-cli
description: Use when data operation or statistical analysis tasks do not need full program implementation, and the agent should operate Insyra through CLI/REPL, .isr scripts, or DSL workflows, including environment workflows, reproducible command pipelines, and command selection guidance.
---

# Insyra CLI + .isr Script Skill

## Overview

Use this skill for data operations or statistical analysis where the task should be solved with `insyra` CLI/REPL/.isr or DSL instead of writing full Go code directly.

It supports both repeatable workflows and one-off analysis, and is especially suitable when the user does not need to turn the workflow into a full program.

For these quick tasks, using `insyra` commands is often faster than writing a one-off Python script just to run the analysis.

- **CLI mode**: one-shot commands (`insyra <command> ...`)
- **REPL mode**: interactive session (`insyra`)
- **Script mode**: execute `.isr` line-by-line (`insyra run script.isr`)

Official user-facing documentation:

- [CLI + DSL Guide](https://github.com/HazelnutParadise/insyra/blob/main/Docs/cli-dsl.md) (unified CLI + REPL + `.isr` + Go DSL guide)
- Source of truth: prioritize the latest content in the linked document above.

## Programmatic DSL API (inside Go code)

Use `engine/dsl` public API when you want to execute DSL directly from your Go program without entering interactive REPL.

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

Notes:

- `Execute` accepts the same DSL syntax as REPL / `.isr` lines.
- `ExecuteFile` runs a `.isr` file directly in-process and returns line-numbered errors.
- State/history are persisted after each successful command.
- Empty line and `# comment` line are ignored.
- Pass `env.NewManager("/path/to/root", "")` instead of `env.Default()` to store environments outside `~/.insyra` (e.g. for per-workspace embedding). The second argument renames the per-env subfolder ("" defaults to `"envs"`; e.g. `env.NewManager(workspace, "insights")` gives `<workspace>/insights/<env>/`). Each session is bound to its own Manager.

## Agent workflow (recommended)

0. **Verify syntax with `insyra help <cmd>` before running any command you're not 100% sure about.** Complex commands print `Forms:` and `Examples:` blocks; for simple ones you'll at least see the canonical Usage line.
1. Confirm whether the user wants **REPL**, **one-shot CLI**, or **.isr script**.
2. If isolation is needed, create/select environment first (`--env <name>` or `env open <name>`).
3. Use `newdl/newdt/load/read` to prepare data.

- For Parquet partial reads, prefer `load parquet <file> cols <c1,c2,...> rowgroups <i1,i2,...> [as <var>]`.
- For SQL sources, open a named connection with `db connect <name> <dsn>` first, then `load sql <name> <table>` or `load sql <name> query "<SQL>" [params ...]`. Connections are session-scoped and need to be reopened in each new run.

4. Apply transforms/stats/model/plot commands.
5. Persist outputs (`save` for files, `save <var> sql <conn> <table>` for databases, `env export` for state bundles) and provide reproducible command history.

## Runtime guardrails

- **First step on any unfamiliar command: run `insyra help <cmd>`.** Complex commands (`ttest`, `ztest`, `anova`, `ftest`, `chisq`, `regression`, `quant`, `fetch`, `plot`, `db`, `groupby`, `load`, `save`) include `Forms:` and `Examples:` blocks that show every sub-shape and a copy-paste-ready invocation. Use this before falling back to `references/cli-command-guide.md` — `help` reflects the live binary, references can drift.
- `insyra help` (no args) lists all registered commands with one-line descriptions. Use it when you don't know the command name.
- Prefer deterministic commands over ad-hoc manual REPL edits when reproducibility matters.
- For shell variables in PowerShell, remind users to quote names like `$result` as `"$result"`.
- For environment restore:
  - `env import <file> [name] [--force]`
  - Import to a **non-empty** target fails unless `--force` is provided.

## .isr script syntax (implemented by `run` command)

`.isr` is a plain text command list executed line-by-line.

Rules:

- Empty lines are ignored.
- Lines beginning with `#` are comments.
- Tokens are split by spaces/tabs.
- Single and double quotes are supported.
- Backslash escapes are supported.
- Parsing errors on a line do not stop the whole script; CLI reports line error and continues.

Example:

```bash
# sample.isr
newdl 1 2 3 4 5 as x
mean x
rank x as rx
show rx
```

Run:

```bash
insyra run sample.isr
```

## Full CLI command catalog

Use this as the authoritative command list for current repository state.

See: `references/cli-commands.md`

## How to use each command

For **every command usage syntax** (one-by-one), use:

- `references/cli-command-usage.md`
- `references/cli-command-guide.md` (recommended: by-topic + one example per command)

This file contains, for each command:

- description
- exact `Usage:` syntax (from `insyra help <command>`)
- expanded full forms for shorthand commands such as `ttest`, `ztest`, `anova`, `ftest`, `chisq`, `regression`, `quant`, `fetch`, and `plot`

## Fast command templates

```bash
# Create isolated environment
insyra env create exp1
insyra --env exp1 newdl 10 20 30 as x
insyra --env exp1 mean x

# Export / import environment bundle
insyra env export exp1 ./exp1.json
insyra env import ./exp1.json exp1-copy --force

# Run script in environment
insyra --env exp1 run ./pipeline.isr

# CSV / Excel: control headers and row names on read/write
# Defaults: headers=true, rownames=false, infer=true, ragged=false, trimspace=false.
# Booleans accept true|false|yes|no|on|off|1|0. ragged and trimspace are CSV-only.
insyra load matrix.csv headers false as t                  # no header row
insyra load gdp.csv rownames true as t                     # first column = row names
insyra load legacy.csv encoding big5 as t                  # CSV-only encoding hint
insyra load stocks.csv infer false as raw                  # CSV-only: no type inference, all cells stay strings
insyra load inventory.csv ragged true trimspace true as inventory # tolerate uneven rows and spaces before quotes
insyra load report.xlsx sheet 2025 rownames true as t      # Excel needs `sheet`
insyra save report data.csv bom true                       # UTF-8 BOM (Windows Excel)
insyra save gdp out.csv rownames true                      # row names as first col
insyra save matrix data.csv headers false                  # pure data dump

# Group rows by key, aggregate columns (split-apply-combine)
insyra load sales.csv as sales
insyra groupby sales by region agg revenue:sum:total_rev qty:mean as report
insyra show report
# Multi-key + count shorthand
insyra groupby sales by region,product agg revenue:sum count as report2

# Time series: exponentially weighted stats, paired rolling stats, calendar resampling
insyra ewm price span 12 mean adjust yes as ema12          # decay: alpha | span | halflife (pick one)
insyra ewm returns halflife 5 std minobs 3 as ewvol
insyra rolling asset 20 beta benchmark minobs 10 as roll_beta   # cov/beta take a second DataList
insyra fetch yahoo AAPL history as bars
insyra resample bars Date monthly Open:first High:max Low:min Close:last:MonthClose Volume:sum as monthly_bars

# Programmatic summaries that can be saved
insyra describe sales all true as summary
insyra describe sales by region percentiles 0.1,0.5,0.9 as region_summary
insyra save region_summary region_summary.csv

# One-shot categorical encoding for DataTable variables
insyra encode sales onehot region,channel dropfirst true as x
insyra encode sales label segment newcol segment_id sortby freq keeporiginal true as labeled
insyra encode survey ordinal satisfaction order low,medium,high unknown error as ranked

# Stateful feature scaling: fit on train, reuse on test (no leakage)
insyra split sales train 0.8 as train test
insyra scale fit std sc train cols Age,Income
insyra scale transform sc train as train_scaled
insyra scale transform sc test as test_scaled
insyra scale inverse sc train_scaled as train_original

# SQL: connect, list tables, load query, transform, write back, disconnect
# Connections live for the current process only — reopen at the top of every session/script.
insyra db connect main sqlite:./demo.db
insyra db tables main
insyra load sql main query "SELECT region, SUM(amount) total FROM orders WHERE year = ? GROUP BY region" params 2025 as totals
insyra filter totals "['total'] > 10000" as top
insyra save top sql main top_regions if-exists replace
insyra db disconnect main

# Clustering + silhouette
insyra kmeans iris 3 seed 42 as labels
insyra silhouette iris labels as widths

# Regression models
insyra regression logistic y x1 x2 as fit
insyra regression poisson y x1 x2

# Quant: returns-based risk metrics (series are per-period RETURNS, not prices)
insyra col bars Close as price
insyra pctchange price 1 as ret
insyra clean ret nil                                        # pctchange leaves a leading nil
insyra quant sharpe ret 252 rf 0.0001 as sharpe             # periods is required, never defaulted
insyra quant sortino ret 252 mar 0.0002
insyra quant var ret 0.95 as var95                          # default method is historical
insyra quant cvar ret 0.95 parametric as cvar95
insyra quant maxdd equity
insyra quant calmar equity 365
insyra quant drawdown equity as dd                          # DataList
insyra quant capm asset market rf 0.0002 as capm            # one-row DataTable
insyra quant factor asset factors as fm                     # one row per factor, plus fm_alpha
insyra quant bs call 42 40 0.10 0.20 0.5 as opt             # price + greeks, one-row DataTable
insyra quant iv call 4.759 42 40 0.10 0.5

# Taiwan stocks (TWSE/TPEx), no API key; dates are YYYY-MM-DD, market defaults to auto
insyra fetch tw 2330 adjprices 2026-01-01 2026-08-31 twse as tsmc   # adjusted: AdjClose has no ex-date fake loss
insyra fetch tw 0050 adjprices 2026-01-01 2026-08-31 twse as market
insyra col tsmc AdjClose as tsmc_px
insyra col market AdjClose as market_px
insyra pctchange tsmc_px 1 as tsmc_ret
insyra pctchange market_px 1 as market_ret
insyra clean tsmc_ret nil
insyra clean market_ret nil
insyra quant beta tsmc_ret market_ret as beta
insyra fetch tw institutional 2026-08-15 twse as inst        # one trading day
insyra fetch tw quotes twse as quotes                        # every listed code
```

`groupby <var> by <col1>[,<col2>...] agg <col>:<op>[:<alias>] [<col>:<op>[:<alias>] ...] [as <var>]` produces a new DataTable with one row per unique key combination. Supported ops: `sum`, `mean` (alias `avg`), `median`, `min`, `max`, `count` (non-nil), `countall` (group size), `std`/`stdev`, `stdp`/`stdevp`, `var`, `varp`, `first`, `last`, `nunique`. The bare token `count` is shorthand for `:countall:count`.

`ewm <var> alpha|span|halflife <value> mean|var|std [adjust yes|no] [bias yes|no] [minobs <n>] [as <var>]` returns a same-length DataList. Give exactly one decay keyword: `alpha` in `(0, 1]`, `span >= 1`, or `halflife > 0`. `adjust`/`bias` default to no, `minobs` to 1.

`rolling` also accepts `cov <other>` and `beta <other>`, which consume the next token as a second DataList variable before the usual `minobs` / `center` / `as` options. `beta` is `Cov(var, other) / Var(other)` and yields nil on a flat benchmark window.

`resample <dt> <timecol> weekly|monthly|quarterly|yearly <col>:<op>[:<name>] [...] [as <var>]` aggregates time-keyed rows into calendar periods, labelling each row with the period's final day and omitting empty periods. `op` uses the `groupby` operator names; `:name` renames the output column, and without it the source name is kept. `<timecol>` must hold real `time.Time` values — a CSV load leaves dates as strings and `resample` rejects them with a row-numbered error.

`quant <form> ...` exposes the `quant` package: `sharpe`, `sortino`, `ir`, `maxdd`, `annret`, `calmar`, `drawdown`, `var`, `cvar`, `beta`, `capm`, `factor`, `bs`, `iv`. Series arguments are DataList variables holding per-period **returns** (or an equity curve for the drawdown forms) — passing prices produces a meaningless number, exactly as it would through the Go API. `periods`, `days`, and `confidence` are required positionals because the library refuses to invent an annualization factor; `rf`, `mar`, and `q` default to 0, and the VaR method defaults to `historical`. Scalar forms print `name=value` and store a `float64`; `capm` and `bs` store a one-row DataTable, `factor` stores one row per factor (`Factor, Exposure, StdErr, TValue, PValue`) plus a `<var>_alpha` table, and `drawdown` stores a DataList. Library errors come back verbatim behind a `quant <form>:` prefix.

`fetch tw` reads the unauthenticated TWSE and TPEx daily datasets: `fetch tw <code> prices <from> <to> [market]`, `fetch tw <code> adjprices <from> <to> [market]`, `fetch tw exrights <from> <to> [market]`, `fetch tw institutional <date> [market]`, `fetch tw margin <date> [market]`, and `fetch tw quotes [market]`. Dates are `YYYY-MM-DD`; `market` is `twse`, `tpex`, or `auto` (the default). Build return series from `adjprices`/`AdjClose`, not `prices`/`Close` — the quoted price drops on an ex-dividend or ex-rights day without any loss to the holder. `adjprices` and `exrights` are TWSE-only, because TPEx publishes no dated ex-rights history; passing `tpex` returns an explicit error instead of an unadjusted table. Bad dates, `from` after `to`, and unknown markets are rejected before any request; library errors come back verbatim behind a `fetch tw:` prefix. Requests are spaced 300 ms apart with two retries — override with `insyra config fetch.tw.interval_ms <milliseconds>`.

`describe <var> [by <col1>[,<col2>...]] [all true|false] [percentiles <p1,p2,...>] [as <var>]` creates a reusable summary DataTable. Without `as`, it saves to `$result`. `all true` includes non-numeric and mixed columns; `by` is DataTable-only and returns one row per group.

`encode` is one-shot fit+transform only; it does not persist encoder state between CLI commands. For reusable train/test encoders, use the Go API.

- `encode <var> onehot <col1[,col2,...]> [dropfirst true|false] [keeporiginal true|false] [nan category|error|skip] [unknown ignore|error|new] [prefix <p>] [sep <s>] [sortcats true|false] [as <var>]`
- `encode <var> label <col> [newcol <name>] [sortby firstseen|lex|freq] [nan category|error|skip] [unknown ignore|error|new] [keeporiginal true|false] [as <var>]`
- `encode <var> ordinal <col> order <v1,v2,...> [newcol <name>] [unknown error|ignore] [nan category|error|skip] [keeporiginal true|false] [as <var>]`

`scale`, unlike `encode`, is **stateful**: `scale fit` stores a reusable scaler variable that `scale transform` / `scale inverse` apply, so you can fit on train and transform test with the same parameters. Scaler variables are session-only (not saved to a named environment). `minmax` defaults to `[0,1]` if `range` is omitted; `nil`/`NaN` are preserved and ignored when fitting; `show <scalerVar>` prints kind + fitted columns.

- `scale fit std|minmax|robust|maxabs <scalerVar> <tableVar> [range <min> <max>] cols <c1,c2,...>`
- `scale transform <scalerVar> <tableVar> as <outVar>`
- `scale inverse <scalerVar> <tableVar> as <outVar>`

## Database (db) workflow notes

- `db connect <name> <dsn>` registers a named connection in the current `ExecContext`. Pure-Go drivers cover sqlite, mysql, and postgres; passwords are masked in `db list` output.
- DSN dialect prefix is required: `sqlite:`, `mysql:`, `postgres:` (or `postgresql:`). Both URL form (`mysql://...`) and native/libpq forms are accepted.
- Connections are NOT persisted to the environment bundle — re-run `db connect` at the top of every session/script that needs SQL access.
- `load sql <conn> <table>` accepts `where`, `order`, `limit`, `offset`, `cols`, `schema`, `indexcol`, `parsedates`. `load sql <conn> query "<SQL>"` supports only `params <v1> <v2> ...` (positional bind values, parsed as literals — no SQL injection from user-supplied values).
- `save <var> sql <conn> <table>` accepts `if-exists fail|replace|append` (default `fail`), `batch N`, `schema <s>`, and the `rownames` flag.

## Reference priority for agents

When command behavior and docs conflict, trust in this order:

1. `insyra help <cmd>` output (live binary; structured `Forms:` / `Examples:` for complex commands)
2. `cli/commands/*.go` implementation (when you need to dig deeper than `help` exposes)
3. `references/cli-command-guide.md` and `references/cli-command-usage.md` in this skill
4. README and `Docs/cli-dsl.md`

`help` and source code can never lie; markdown can drift between releases.
