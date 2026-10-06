## ADDED Requirements

### Requirement: Creating an environment is safe from several callers at once

Of several `Create` calls for the same environment running at once, exactly one SHALL succeed and the others SHALL report that the environment already exists. Creating an environment SHALL NOT overwrite a file already in its folder. `EnsureDefaultEnvironment` SHALL succeed when another caller creates `default` while it runs.

#### Scenario: Concurrent Create
- **WHEN** eight goroutines call `Create("shared")` on the same Manager at once
- **THEN** exactly one returns nil

#### Scenario: A state saved before the default files are written
- **WHEN** an environment's folder already holds a `state.json` with a variable, and the default files are written into it
- **THEN** the `state.json` still holds that variable, and `history.txt` and `config.json` exist

#### Scenario: Concurrent EnsureDefaultEnvironment
- **WHEN** four goroutines call `EnsureDefaultEnvironment` on a Manager whose root is empty
- **THEN** every call returns nil
