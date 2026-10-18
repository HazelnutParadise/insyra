# Proposal: cli-env-saves-merge

## Why

Each CLI command read the whole `state.json`, ran, and wrote the whole file back. Two `insyra` commands on one environment at the same time each saved their own copy, so the later save erased the variables the other had created. Measured on this line: twelve processes each adding one variable ended with one.

## What Changes

- A save that follows a restore in the same process writes only what that process changed since (variables it added, changed or removed) on top of what the file holds at that moment, under a short lock held for the save alone.
- The temporary file has a name of its own per writer, so two saves no longer write the same `state.json.tmp`.
- Not changed: a save with no restore before it replaces the file as before; commands are not serialised, only the save is.

## Capabilities

### Modified Capabilities

- `env-management`: concurrent commands on one environment keep each other's variables.

## Impact

`cli/env/state.go`, `cli/env/state_lock.go`, `cli/env/manager.go`, `cli/env/state_concurrent_test.go`, `Docs/cli-dsl.md`, both CHANGELOGs, `AGENTS.md` follow-up removed.
