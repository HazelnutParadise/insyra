# dsl-layering Specification

## Purpose
Where the insyra command language lives and which way its dependencies point: below both the CLI and `engine/dsl`, so a Go program running DSL lines builds none of the shell, while the public `cli/` names it had keep working.
## Requirements
### Requirement: engine/dsl does not depend on the CLI

`engine/dsl` SHALL NOT depend, directly or transitively, on any package under `cli/`, on `github.com/spf13/cobra`, or on `github.com/ergochat/readline`. The command language (its commands and registry, environments, output styling and session) SHALL live outside `cli/`, and both `cli/` and `engine/dsl` SHALL import it.

#### Scenario: Listing engine/dsl's dependencies
- **WHEN** `go list -deps` runs on `engine/dsl`
- **THEN** no listed package is under `github.com/HazelnutParadise/insyra/cli`, `github.com/spf13/cobra` or `github.com/ergochat/readline`

### Requirement: Public names survive the move

The exported API of `engine/dsl` SHALL be unchanged: `NewSession` SHALL take `*cli/env.Manager`, and `Session.Context` SHALL return `*cli/commands.ExecContext`. `cli/commands`, `cli/env` and `cli/style` SHALL keep every exported name, as aliases of the moved types and functions that call the moved ones. `cli/repl.DSLSession` and `NewDSLSession` SHALL stay for one release as Deprecated names of `engine/dsl`'s `Session` and `NewSession`.

#### Scenario: A program written against the old layout
- **WHEN** a program calls `dsl.NewSession(env.Default(), "default", nil)` with `env` from `cli/env`, runs `newdl 1 2 3 as x`, and reads `session.Context().Vars`
- **THEN** it compiles unchanged and `x` is a DataList of three values

#### Scenario: The deprecated session names
- **WHEN** a caller uses `repl.NewDSLSession` with a Manager
- **THEN** it gets the same session `engine/dsl.NewSession` returns, and a nil Manager is refused as before

### Requirement: The shell is built in cli/commands

`BuildCobraCommands` SHALL stay in `cli/commands` and build one shell command per registered command, in name order, from the registry the command language reads.

#### Scenario: Every registered command has a shell command
- **WHEN** `BuildCobraCommands` runs
- **THEN** it returns one command per name `SnapshotRegistry` lists, in the same order

### Requirement: A program needs nothing from cli/ to use the command language

`engine/dsl` SHALL provide `Manager`, `NewManager`, `DefaultManager`, and the types the Manager's methods use (`EnvironmentInfo`, `GlobalConfig`, `State`, `SerializedVariable`, `UnsavedVariable`), as the same types `cli/env` names. `DefaultManager` SHALL return a new Manager rooted at `<UserHomeDir>/.insyra`, with environments in `envs/`, on every call. The same names in `cli/env` SHALL carry a `Deprecated:` notice pointing at `engine/dsl` and stay for one release; `cli/env.Default`, `ConfigKeys` and `ExportPayload` SHALL NOT be deprecated.

#### Scenario: A workspace-scoped session from engine/dsl alone
- **WHEN** a program importing only `engine/dsl` creates `NewManager(root, "insights")`, creates the environment `analysis`, opens a session on it and runs `newdl 1 2 3 as x`
- **THEN** `root/insights/analysis/state.json` exists and `List` reports one variable in `analysis`

#### Scenario: Default managers are independent
- **WHEN** a program calls `DefaultManager()` twice and moves the first with `SetBasePath`
- **THEN** the second is still at `<UserHomeDir>/.insyra/envs`, and the two are different values

#### Scenario: A session on an environment that does not exist
- **WHEN** `NewSession` is called with an environment other than `default` that has not been created
- **THEN** it returns an error

