# Proposal: cli-unreadable-state-stops

## Why

When `state.json` could not be read, the CLI started from an empty variable map without saying so, and the save after the next command wrote that map over the file, so every variable in the environment was lost. Measured: after `newdl 1 2 3 as x`, truncating `state.json` and running `newdl 9 as y` left a file holding only `y`. `env open` kept the previous environment's variables and the next save wrote them into the environment just opened.

## What Changes

- A `state.json` that exists but cannot be read stops the command, the REPL, the Go session and `env open` with an error naming the file and the cause. Nothing runs and the file is not touched. A missing `state.json` still starts an empty environment.
- `env open` reads the new environment's variables before switching, so a failure leaves the current environment as it was.

## Capabilities

### Modified Capabilities

- `env-management`: see What Changes.

## Impact

`cli/env/state_open.go`, `cli/root.go`, `cli/repl/`, `cli/commands/env.go`, `Docs/cli-dsl.md`, both CHANGELOGs, `AGENTS.md` follow-up removed.
