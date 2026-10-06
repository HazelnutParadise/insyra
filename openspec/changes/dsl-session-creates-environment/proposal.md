# Proposal: dsl-session-creates-environment

## Why

A program names the environment its session uses in code, and on its first run that environment does not exist yet. `engine/dsl.NewSession` created only `default` and failed on any other name with `environment does not exist`, so every program had to check with `mgr.Exists` and call `mgr.Create` first. The example in `Docs/cli-dsl.md` did not, and failed as written. Idensyra, the one embedder found, wrote its own `getOrCreate` for the same reason. The owner chose on 2026-10-06 to have `NewSession` create a missing environment and keep the `insyra` command strict.

Testing the new behaviour from two goroutines found that creating an environment was not safe to do concurrently, before this change as well:

- `Create` checked for the folder, then made it with `MkdirAll` and wrote the default files, so two `Create`s of one name could both succeed (measured: 2 of 8 in a round), and the later one rewrote `state.json` with an empty state, which could wipe the variables the earlier one's session had already saved.
- `EnsureDefaultEnvironment`, which every `NewSession` and every `insyra` command calls, failed with `environment already exists: default` when another caller created `default` between its check and its `Create`.

## What Changes

- `NewSession` creates the environment it is given when it does not exist and opens it when it does. A name an environment may not have is still an error. The deprecated `cli/repl.NewDSLSession`, the same function, does the same.
- The `insyra` command does not change: `--env name` and `env open name` still fail on a missing environment, so a name mistyped in a terminal is not turned into a new, empty environment.
- `Create` makes the environment's folder with `os.Mkdir`, which fails for a folder that is already there, so of concurrent `Create`s exactly one succeeds and the others report that it exists. It writes each default file only where none exists.
- `EnsureDefaultEnvironment` and `NewSession` treat a `Create` that lost to another caller as success when the environment is there afterwards.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `dsl-layering`: `NewSession` creates a missing environment.
- `env-management`: creating an environment is safe from several callers at once.

## Impact

- Code: `internal/dsl/session.go` (`NewSession`), `internal/dsl/env/manager.go` (`Create`, `EnsureDefaultEnvironment`, `writeDefaultFiles`), `engine/dsl/dsl.go` (doc).
- Tests: `engine/dsl/manager_test.go`, `internal/dsl/env/create_race_test.go`.
- Docs: `Docs/cli-dsl.md`, `engine/README.md`, `CHANGELOG.md`, `CHANGELOG_TW.md` (and the `engine-dsl-manager` entry that said a session opens only an existing environment), `delivery-status.md`.
