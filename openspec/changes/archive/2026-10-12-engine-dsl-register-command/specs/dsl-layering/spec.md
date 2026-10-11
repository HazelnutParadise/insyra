## ADDED Requirements

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
