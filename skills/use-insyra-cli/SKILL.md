---
name: use-insyra-cli
description: Use when a data operation or statistical analysis can be done without writing a full program, by driving Insyra through its CLI, REPL, `.isr` scripts or the Go DSL. Teaches when to use each mode, how sessions and environments behave, how to keep the work reproducible, and how to find the exact commands for the installed version from the binary and its docs. It deliberately does not list the commands.
---

# Insyra CLI, REPL and `.isr` scripts

Insyra has one command language, and it runs four ways: one-shot (`insyra <command> ...`), the REPL (`insyra`), `.isr` scripts (`insyra run file.isr`), and Go code through `engine/dsl`. Reach for it when the task is analysis rather than a program: load a file, clean it, run a test or a model, save a result. That is often faster than writing a throwaway Python script.

Write Go instead (the `insyra` skill) when the result has to be a program, run inside a service, or needs logic the commands cannot express. A common path is to prototype with commands and then port the working steps to Go.

This skill does not list the commands, on purpose. The installed binary knows exactly which commands and options it has, and a list here would describe some other version.

## Find commands in the binary, not in memory

- `insyra help` lists every command the installed binary has, with a one-line description.
- `insyra help <command>` prints the command's usage line. Commands with several shapes also print `Forms:` and `Examples:`. Run it before any command you have not already checked in this session, and follow the usage line exactly: `[...]` is optional, `<...>` is a value you supply, `a|b` is a choice.
- `insyra version` prints the installed release, for example `insyra v0.3.3 (Huashan)`. The user guide for that release is `Docs/cli-dsl.md` at the matching tag, `https://github.com/HazelnutParadise/insyra/blob/v0.3.3/Docs/cli-dsl.md`. It holds the full command index, the commands grouped by topic, worked workflows and the syntax rules. `go mod download -json github.com/HazelnutParadise/insyra@v0.3.3` prints the module's `Dir`, from the module cache when the release is already there (as it is after `go install`), and the same file is at `<Dir>/Docs/cli-dsl.md`.
- When sources disagree, trust `insyra help`, then the command's source (`cli/commands/*.go` at that tag), then `Docs/cli-dsl.md`. Prose can drift; the registry the binary prints from cannot.
- To find a command by what it does, read the `Command Groups` section of `Docs/cli-dsl.md`, or scan `insyra help`.
- A command runs a library function, so the library's documentation explains its results. For example, `Docs/stats.md` gives a test's assumptions and the meaning of each field it reports. The `insyra` skill tells you how to find those pages.

## How a session works

- **Everything is a named variable.** A command that creates or transforms data takes `as <var>`. Without it, the result goes to `$result`, which the next such command overwrites. Name everything you intend to use again.
- **Variables live in an environment.** An environment keeps its variables in `~/.insyra/envs/<name>/state.json` and its history in `history.txt`. The environment is `default` unless you pass `--env <name>`, which must name an existing environment (`env create <name>` makes one). Inside the REPL or a script, `env open <name>` switches; run as a one-shot command it opens the REPL instead.
- **A one-shot command reloads the environment from disk, and the round trip loses things.** `insyra newdl 1 2 3 as x` followed by a separate `insyra mean x` works. But a table comes back with its columns in alphabetical order and its dates as text, and a variable the file cannot hold, such as a fitted scaler, is gone. Column letters then point at different columns. Do multi-step work in the REPL or in a script, where variables stay in memory, and use separate one-shot commands only for independent steps.
- **Connections do not persist.** A database connection made with `db connect` lives only in the running process. Reconnect at the top of every script or session that needs it.
- **Scripts keep going after an error.** An `.isr` file holds one command per line. A line starting with `#` is a comment, but `#` later in a line is data. Quotes and backslash escapes work. A failing line is reported with its line number and the next line runs, so read the output instead of assuming the script succeeded.
- **The Go DSL is the same language.** `engine/dsl` runs the same lines inside a program and saves state after each successful command. Unlike `run`, its `ExecuteFile` stops at the first failing line. Use it when a Go program should hand the user a scriptable surface.

## Principles

1. **Choose the mode on purpose.** Explore in the REPL. Anything you will rerun or hand over belongs in an `.isr` script. Use one-shot commands for a single step in a shell pipeline, and `engine/dsl` to embed the language in a program. Ask if the user's intent is unclear.
2. **Isolate the work.** Create an environment for each task, so you neither read nor overwrite someone's variables in `default`. `env export` hands the state over. Importing into an environment that already has variables, history or config needs `--force`, and that replaces what was there; treat it as destructive and confirm first.
3. **Prefer reproducible steps to interactive fixes.** Results should come from commands someone can rerun, not from edits made by hand in the REPL. Keep the commands you ran, or write them as a script.
4. **Look before you trust.** After every load or transform, `show` or `describe` the result. Check the shape, the column types, whether the header was read as data, and whether numbers arrived as text, before you analyse anything.
5. **Read only what you need.** Some sources can be read in part, such as chosen columns and row groups of a Parquet file, or a query against a database. Check `insyra help load` before loading everything and filtering afterwards.
6. **Save results explicitly, and say where.** Write files with `save`, write tables to a database with `save <var> sql ...`, and export the session with `env export`. Tell the user the path or table.
7. **Mind the shell.** bash, zsh and PowerShell all expand `$result` inside double quotes or bare, so pass it in single quotes: `'$result'`. Quote any token that contains spaces.
8. **Read errors, then check usage.** An error says what was wrong with the invocation, and a script prefixes it with the line number. Compare the invocation with `insyra help <command>` and fix it instead of retrying variations.

## Workflow

1. Run `insyra version` and `insyra help`. Confirm the commands you plan to use exist.
2. Choose the mode and the environment.
3. Load, inspect, transform, analyse, save. Run `insyra help <command>` for each command you have not yet checked.
4. Report the exact commands or the script you ran, and where every output went.
