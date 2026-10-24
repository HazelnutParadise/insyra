## ADDED Requirements

### Requirement: NewSession creates a missing environment

`NewSession` SHALL create the environment it is given when it does not exist, and open it when it does.

#### Scenario: A session on an environment that does not exist
- **WHEN** `NewSession` is called with an environment that has not been created, runs `newdl 1 2 3 as x`, and a second `NewSession` opens the same environment
- **THEN** the environment exists after the first call, and the second session holds `x`

#### Scenario: Two sessions on a missing environment at once
- **WHEN** two goroutines call `NewSession` on the same missing environment at the same time
- **THEN** both succeed

### Requirement: NewSession refuses a name outside the environments directory

`NewSession` SHALL return an error, and create nothing, for an environment name that resolves outside the environments directory.

#### Scenario: A name that escapes the environments directory
- **WHEN** `NewSession` is called with `../escape`
- **THEN** it returns an error and no environment is created

#### Scenario: A nested name
- **WHEN** `NewSession` is called with `team/run`
- **THEN** the environment is created at `envs/team/run` and the session opens it

### Requirement: NewSession returns a failed creation

`NewSession` SHALL return the error from creating a missing environment, unless another caller has completed that environment in the meantime.

#### Scenario: An existing environment without its default files
- **WHEN** `NewSession` is called on an environment whose folder exists but lacks `state.json`, `history.txt` or `config.json`, and nothing completes it
- **THEN** it returns an error

#### Scenario: Creation fails for another reason
- **WHEN** `NewSession` is called on a missing environment and the environments directory cannot be written
- **THEN** it returns the error from the failed creation
