## ADDED Requirements

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
