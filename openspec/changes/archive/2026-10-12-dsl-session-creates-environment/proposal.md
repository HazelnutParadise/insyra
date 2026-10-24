# Proposal: dsl-session-creates-environment

## Why

A program names the environment its session uses in code, and on its first run that environment does not exist yet. `engine/dsl.NewSession` created only `default` and failed on any other name with `environment does not exist`, so every program had to check with `mgr.Exists` and call `mgr.Create` first. The example in `Docs/cli-dsl.md` did not, and failed as written. Idensyra, the one embedder found, wrote its own `getOrCreate` for the same reason. The owner chose on 2026-10-06 to have `NewSession` create a missing environment and keep the `insyra` command strict.

Testing the new behaviour from two goroutines found that creating an environment was not safe to do concurrently, before this change as well:

- `Create` checked for the folder, then made it with `MkdirAll` and wrote the default files, so two `Create`s of one name could both succeed (measured: 2 of 8 in a round), and the later one rewrote `state.json` with an empty state, which could wipe the variables the earlier one's session had already saved.
- `EnsureDefaultEnvironment`, which every `NewSession` and every `insyra` command calls, failed with `environment already exists: default` when another caller created `default` between its check and its `Create`.

## What Changes

- `NewSession` creates the environment it is given when it does not exist and opens it when it does. A name that resolves outside the environments directory (an absolute name, or one that escapes through `..`) is still an error. A nested name such as `team/run` is created.
- `NewSession` returns the error from creating the environment, unless another caller completed it first. It waits up to about one second for the default files of an environment another caller is creating, and returns the original error if they do not appear.
- The `insyra` command does not change: `--env name` and `env open name` still fail on a missing environment, so a name mistyped in a terminal is not turned into a new, empty environment.
- `Create` makes the environment's folder with `os.Mkdir`, which fails for a folder that is already there, so of concurrent `Create`s exactly one succeeds and the others report that it exists. Parent folders of a nested name are still made with `MkdirAll`. `Create` writes each default file only where none exists.
- `EnsureDefaultEnvironment` treats a `Create` that lost to another caller as success when the environment is there afterwards.

Left on 0.4:

- `cli/repl.NewDSLSession` still refuses an environment that does not exist on this line. 0.4 makes it create one too, since it deprecates that function.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `dsl-layering` (created by `engine-dsl-manager`): `NewSession` creates a missing environment.
- `env-management`: creating an environment is safe from several callers at once.

## Impact

- Code: `cli/env/manager.go` (`Create`, `writeDefaultFiles`, `EnsureDefaultEnvironment`), `engine/dsl/dsl.go` (`NewSession`).
- Tests: `engine/dsl/manager_test.go`, `cli/env/create_race_test.go`.
- Docs: `Docs/cli-dsl.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`.
