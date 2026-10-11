# Proposal: engine-dsl-manager

## Why

`engine/dsl` has been the documented way to run the command language from Go since it was added (d6d307ac, 2026-03-01): that commit wrote its section in `engine/README.md` and told agents in `skills/use-insyra-cli` to use it, and its `NewSession(envName, output)` needed nothing from `cli/`. #160 (2026-05-09) changed it to `NewSession(mgr *env.Manager, ...)` with the Manager from `cli/env`, so from then on a program using the documented entry point had to import a CLI package as well. `dsl-outside-cli` (#260) removed `engine/dsl`'s own dependency on `cli/` but kept that signature.

The one embedder found, Idensyra, shows the cost. It imports `cli/commands`, builds an `ExecContext` and calls `commands.Dispatch` itself, and keeps its own copy of the tokenizer and of a `state.json` format insyra has since replaced.

## What Changes

- `engine/dsl` gains `Manager`, `NewManager` and `DefaultManager`, and the types the Manager's methods use: `EnvironmentInfo`, `GlobalConfig`, `State`, `SerializedVariable`, `UnsavedVariable`. They are aliases of the types `cli/env` already names, so existing code keeps compiling.
- `DefaultManager()` returns a new Manager rooted at `<UserHomeDir>/.insyra` on every call, the place the `insyra` command uses. A Manager holds only its root and subfolder name, so a new one reads and writes the same files as the shared `cli/env.Default()`, and moving one with `SetBasePath` moves no other. The owner chose this over returning the shared instance on 2026-10-05.
- `Docs/cli-dsl.md`'s `engine/dsl` examples use `NewManager` and `DefaultManager`.

Left on 0.4:

- The seven `cli/env` names above are not marked `Deprecated:` on this line. 0.4 marks them under the one-name rule (#211).
- The nil-manager error in `cli/repl/api.go` still says `env.Default()`. 0.4 names `dsl.DefaultManager()` and `dsl.NewManager(...)` instead.

Registering a command from `engine/dsl` is the next change, `engine-dsl-register-command`.

## Capabilities

### New Capabilities

- `dsl-layering`: `engine/dsl` provides the Manager and its types, so a program needs nothing under `cli/`.

### Modified Capabilities

(none)

## Impact

- Code: new `engine/dsl/manager.go`.
- Tests: `engine/dsl/manager_test.go`.
- Docs: `Docs/cli-dsl.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`.
