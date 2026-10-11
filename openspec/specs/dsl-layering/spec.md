# dsl-layering Specification

## Purpose
What a Go program can do with the command language through `engine/dsl` alone, without importing `cli/`: managing environments, opening sessions that create a missing environment, and registering its own commands.

## Requirements

### Requirement: A program needs nothing from cli/ to use the command language

`engine/dsl` SHALL provide `Manager`, `NewManager`, `DefaultManager` and the types the Manager's methods use, so a program needs nothing under `cli/` to run the command language.

#### Scenario: A workspace-scoped session from engine/dsl alone
- **WHEN** a program importing only `engine/dsl` creates `NewManager(root, "insights")`, creates the environment `analysis`, opens a session on it and runs `newdl 1 2 3 as x`
- **THEN** `root/insights/analysis/state.json` exists and `List` reports one variable in `analysis`

### Requirement: DefaultManager returns an independent Manager

`DefaultManager` SHALL return a new Manager rooted at `<UserHomeDir>/.insyra`, with environments in `envs/`, on every call.

#### Scenario: Default managers are independent
- **WHEN** a program calls `DefaultManager()` twice and moves the first with `SetBasePath`
- **THEN** the second is still at `<UserHomeDir>/.insyra/envs`, and the two are different values

### Requirement: A program registers its own command through engine/dsl

`engine/dsl` SHALL provide `Register` with the types and functions a registration needs, so a command registered there is available in every session in the process.

#### Scenario: A registered command in a session
- **WHEN** a program registers `zzdouble` through `engine/dsl`, opens a session and runs `zzdouble x as y`
- **THEN** the command's output reaches the session's output, and `y` is saved in the environment

#### Scenario: A handler without Run
- **WHEN** a program registers a handler with no `Run` function
- **THEN** `Register` returns an error

### Requirement: Register refuses a name already taken

`Register` SHALL refuse a name that is already registered, built-in commands included, and leave the registered command unchanged.

#### Scenario: A taken name
- **WHEN** a program registers a command named `newdl`
- **THEN** `Register` returns an error

#### Scenario: A duplicate registration keeps the first handler
- **WHEN** a program registers `zzdouble`, then registers `zzdouble` again with a different `Run`
- **THEN** `Register` returns an error for the second, and running `zzdouble` still calls the first `Run`

### Requirement: A declared argument limit is enforced before Run

A command registered with an argument limit SHALL be refused, before its `Run` is called, when it is given more arguments than the limit allows.

#### Scenario: An argument past the declared count
- **WHEN** `zzdouble`, declared with `MaxArgs(1).WithAlias()`, is run as `zzdouble x y z`
- **THEN** `Execute` returns an error and `Run` is not called

### Requirement: Register refuses a negative argument limit

`Register` SHALL refuse a handler whose argument limit or form position is negative, without registering the command.

#### Scenario: A negative limit
- **WHEN** a program registers a handler declared with `MaxArgs(-1)`
- **THEN** `Register` returns an error and the command is not registered

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
