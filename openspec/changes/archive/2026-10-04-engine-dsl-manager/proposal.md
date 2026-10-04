# Proposal: engine-dsl-manager

## Why

`engine/dsl` has been the documented way to run the command language from Go since it was added (d6d307ac, 2026-03-01): that commit wrote its section in `engine/README.md` and told agents in `skills/use-insyra-cli` to use it, and its `NewSession(envName, output)` needed nothing from `cli/`. #160 (2026-05-09) changed it to `NewSession(mgr *env.Manager, ...)` with the Manager from `cli/env`, so from then on a program using the documented entry point had to import a CLI package as well. `dsl-outside-cli` (#260) removed `engine/dsl`'s own dependency on `cli/` but kept that signature.

The one embedder found, Idensyra, shows the cost. It imports `cli/commands`, builds an `ExecContext` and calls `commands.Dispatch` itself, and keeps its own copy of the tokenizer and of a `state.json` format insyra has since replaced.

## What Changes

- `engine/dsl` gains `Manager`, `NewManager` and `DefaultManager`, and the types the Manager's methods use: `EnvironmentInfo`, `GlobalConfig`, `State`, `SerializedVariable`, `UnsavedVariable`. They are aliases of the types `cli/env` already names, so existing code keeps compiling.
- `DefaultManager()` returns a new Manager rooted at `<UserHomeDir>/.insyra` on every call, the place the `insyra` command uses. A Manager holds only its root and subfolder name, so a new one reads and writes the same files as the shared `cli/env.Default()`, and moving one with `SetBasePath` moves no other. The owner chose this over returning the shared instance on 2026-10-05.
- Under the one-name rule of #211, `cli/env`'s `Manager`, `NewManager`, `EnvironmentInfo`, `GlobalConfig`, `State`, `SerializedVariable` and `UnsavedVariable` are Deprecated for one release. `Default`, `ConfigKeys` and `ExportPayload` have no counterpart in `engine/dsl` and stay.
- `SetBasePath` and `SetEnvsDirName` stay public, as the owner chose on 2026-10-05. `NewManager`'s doc said both settings were fixed at construction, which those two methods contradict; it now says they move the Manager and must not be used while an environment is in use.
- `engine/dsl`'s docs say what was left unsaid: a session opens an existing environment (only `default` is created), and a `Session` is not safe for concurrent use. Checking the first found the `engine/dsl` example in `Docs/cli-dsl.md` opening `demo` without creating it, which fails with `environment does not exist: demo`; the example now creates it.
- `NewSession`'s nil-manager error names `dsl.DefaultManager()` and `dsl.NewManager(...)` instead of `env.Default()`.

Out of scope: registering a custom command from `engine/dsl`. It needs `cli/commands.Register` today, which pulls in Cobra; where that API belongs is undecided.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `dsl-layering`: `engine/dsl` provides the Manager and its types, so a program needs nothing under `cli/`.

## Impact

- Code: new `engine/dsl/manager.go`; `engine/dsl/dsl.go` (docs, the `Manager` name in `NewSession`), `cli/env/env.go` (Deprecated notices), `cli/repl/session.go` (names `dsl.Manager`), `internal/dsl/session.go` (error text), `internal/dsl/env/manager.go` (`NewManager` doc).
- Tests: `engine/dsl/manager_test.go`, `TestNamesEngineDSLNowProvidesAreDeprecated` in `cli/env/deprecated_wrappers_test.go`; tests that used the deprecated names move to `engine/dsl`'s.
- Docs: `Docs/cli-dsl.md`, `engine/README.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`, `AGENTS.md`, `delivery-status.md`.
