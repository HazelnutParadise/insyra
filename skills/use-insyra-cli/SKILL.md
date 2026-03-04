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

Notes:

- `Execute` accepts the same DSL syntax as REPL / `.isr` lines.
- `ExecuteFile` runs a `.isr` file directly in-process and returns line-numbered errors.
- State/history are persisted after each successful command.
- Empty line and `# comment` line are ignored.

## Agent workflow (recommended)

1. Confirm whether the user wants **REPL**, **one-shot CLI**, or **.isr script**.
2. If isolation is needed, create/select environment first (`--env <name>` or `env open <name>`).
3. Use `newdl/newdt/load/read` to prepare data.

- For Parquet partial reads, prefer `load parquet <file> cols <c1,c2,...> rowgroups <i1,i2,...> [as <var>]`.

4. Apply transforms/stats/model/plot commands.
5. Persist outputs (`save`, `env export`) and provide reproducible command history.

## Runtime guardrails

- Prefer deterministic commands over ad-hoc manual REPL edits when reproducibility matters.
- For shell variables in PowerShell, remind users to quote names like `$result` as `"$result"`.
- Use `help` (or `help <command>`) when syntax is uncertain.
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
```

## Reference priority for agents

When command behavior and docs conflict, trust in this order:

1. `cli/commands/*.go` implementation
2. `insyra help` output
3. README command examples
