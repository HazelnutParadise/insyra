## MODIFIED Requirements

### Requirement: A program needs nothing from cli/ to use the command language

`engine/dsl` SHALL provide `Manager`, `NewManager`, `DefaultManager`, and the types the Manager's methods use (`EnvironmentInfo`, `GlobalConfig`, `State`, `SerializedVariable`, `UnsavedVariable`), as the same types `cli/env` names. `DefaultManager` SHALL return a new Manager rooted at `<UserHomeDir>/.insyra`, with environments in `envs/`, on every call. The same names in `cli/env` SHALL carry a `Deprecated:` notice pointing at `engine/dsl` and stay for one release; `cli/env.Default`, `ConfigKeys` and `ExportPayload` SHALL NOT be deprecated. `NewSession` SHALL create the environment it is given when it does not exist and open it when it does, and SHALL return an error for a name an environment may not have.

#### Scenario: A workspace-scoped session from engine/dsl alone
- **WHEN** a program importing only `engine/dsl` creates `NewManager(root, "insights")`, opens a session on the environment `analysis` and runs `newdl 1 2 3 as x`
- **THEN** `root/insights/analysis/state.json` exists and `List` reports one variable in `analysis`

#### Scenario: Default managers are independent
- **WHEN** a program calls `DefaultManager()` twice and moves the first with `SetBasePath`
- **THEN** the second is still at `<UserHomeDir>/.insyra/envs`, and the two are different values

#### Scenario: A session on an environment that does not exist
- **WHEN** `NewSession` is called with an environment that has not been created, runs `newdl 1 2 3 as x`, and a second `NewSession` opens the same environment
- **THEN** the environment exists after the first call, and the second session holds `x`

#### Scenario: A name an environment may not have
- **WHEN** `NewSession` is called with `a/b`
- **THEN** it returns an error and no environment is created

#### Scenario: Two sessions on a missing environment at once
- **WHEN** two goroutines call `NewSession` on the same missing environment at the same time
- **THEN** both succeed
