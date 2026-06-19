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

- **First step on any unfamiliar command: run `insyra help <cmd>`.** Complex commands (`ttest`, `ztest`, `anova`, `ftest`, `chisq`, `regression`, `fetch`, `plot`, `db`, `groupby`, `load`, `save`) include `Forms:` and `Examples:` blocks that show every sub-shape and a copy-paste-ready invocation. Use this before falling back to `references/cli-command-guide.md` — `help` reflects the live binary, references can drift.
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
- expanded full forms for shorthand commands such as `ttest`, `ztest`, `anova`, `ftest`, `chisq`, `regression`, `fetch`, and `plot`

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
# Defaults: headers=true, rownames=false. Booleans accept true|false|yes|no|on|off|1|0.
insyra load matrix.csv headers false as t                  # no header row
insyra load gdp.csv rownames true as t                     # first column = row names
insyra load legacy.csv encoding big5 as t                  # CSV-only encoding hint
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
```

`groupby <var> by <col1>[,<col2>...] agg <col>:<op>[:<alias>] [<col>:<op>[:<alias>] ...] [as <var>]` produces a new DataTable with one row per unique key combination. Supported ops: `sum`, `mean` (alias `avg`), `median`, `min`, `max`, `count` (non-nil), `countall` (group size), `std`/`stdev`, `stdp`/`stdevp`, `var`, `varp`, `first`, `last`, `nunique`. The bare token `count` is shorthand for `:countall:count`.

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
