## ADDED Requirements

### Requirement: Concurrent Create calls for one environment have one winner

`Create` SHALL succeed for exactly one of several concurrent calls for the same environment name.

#### Scenario: Concurrent Create
- **WHEN** eight goroutines call `Create("shared")` on the same Manager at once
- **THEN** exactly one returns nil, and the others report that the environment already exists

### Requirement: Creating an environment never overwrites a file

Creating an environment SHALL NOT overwrite a file already in its folder.

#### Scenario: A state saved before the default files are written
- **WHEN** an environment's folder already holds a `state.json` with a variable, and the default files are written into it
- **THEN** the `state.json` still holds that variable, and `history.txt` and `config.json` exist

### Requirement: EnsureDefaultEnvironment tolerates a concurrent creator

`EnsureDefaultEnvironment` SHALL succeed when another caller creates `default` while it runs.

#### Scenario: Concurrent EnsureDefaultEnvironment
- **WHEN** four goroutines call `EnsureDefaultEnvironment` on a Manager whose root is empty
- **THEN** every call returns nil
