# Proposal: cli-overwrite-and-delete-documented

## Why

Several CLI commands replace or remove what is already there without asking, and neither `insyra help` nor `Docs/cli-dsl.md` says so (#320, api-review CLI-11). Measured on 2026-10-03 with the CLI built from `0.4` at 61e5b0e9, with `HOME` in a temporary directory:

- `insyra --env work env delete default` succeeds and removes the default environment's variables and history; the next command recreates it empty. Only the environment in use is protected.
- `save dt out.csv` run twice replaces the file the second time, as `save` to JSON and Parquet does.
- `plot line dt` without `save` writes `line.html` into the working directory, replacing any file of that name.
- `env export <name> <file>` replaces `<file>`.
- `env clear` empties the history too unless given `--keep-history`.
- `env open` replaces the session's variables with the opened environment's; a variable the environment could not store (a `regression` result) is gone.

The existing `env-management` spec even says `env delete` asks for confirmation, which it never did.

## What Changes

- `env delete default` is refused unless `--force` is given, the convention `env import … --force` already uses. The refusal says what deleting `default` loses and how to go ahead. Every other environment deletes as before, and `--force` does not lift the refusal to delete the environment in use. An unknown flag after `env delete` is an error. In one-shot use `--force` reaches `env delete` as it reaches `env import`: `env`'s `CommandFlag` for it (introduced by `cli-env-one-name`) names both forms, `Form: "import|delete"`, and a `Form` may now list several words separated by `|`.
- `env`'s, `save`'s and `plot`'s help (`Forms`) and `Docs/cli-dsl.md` say which of their forms replace or remove existing data: `env delete`, `env clear` (history included unless `--keep-history`), `env open` (the session's unsaved variables), `env export` (the output file), `env import --force`, `save` to a CSV, JSON or Parquet file, and `plot` with or without `save`. `save` to Excel already refuses to replace a sheet unless asked, and the docs say so already.
- Nothing else gains a confirmation prompt: the CLI runs from scripts and one-shot shell commands, where a prompt would block, and `--force` is the existing way to say "I mean it".
- Both CHANGELOGs, `api-review.md`, `delivery-status.md` and the `use-insyra-cli` skill's principle on isolating work are updated.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `env-management`: `env delete` deletes without a prompt, refuses the environment in use, and refuses `default` without `--force`.
- `dsl-commands`: commands that replace or remove existing data say so in their help and in `Docs/cli-dsl.md`.
- `command-registry`: a one-shot flag's `Form` may list several words.

## Impact

- Code: `internal/dsl/commands/env.go`, `internal/dsl/commands/registry.go` (the `CommandFlag.Form` comment), `cli/commands/cobra.go` (matching several words), `internal/dsl/AGENTS.md`, `internal/dsl/commands/save.go` and `internal/dsl/commands/plot.go` (`Forms` only).
- Tests: `internal/dsl/commands/env_delete_test.go`, `cli/commands/cobra_test.go`, a one-shot case in `cli/root_flags_test.go`.
- Docs: `Docs/cli-dsl.md`, `skills/use-insyra-cli/SKILL.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`, `api-review.md`, `delivery-status.md`.
- `env delete default` without `--force` used to succeed and now fails. No library change, no new dependency.
