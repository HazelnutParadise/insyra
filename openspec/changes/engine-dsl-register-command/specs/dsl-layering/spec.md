## ADDED Requirements

### Requirement: A program registers its own command through engine/dsl

`engine/dsl` SHALL provide `Register`, `CommandHandler`, `ExecContext`, `CommandFlag`, `ArgLimit`, `MaxArgs`, `FormArgs`, `FormArgsAt` and `OpenArgs`, as the same types and functions the command language's registry uses. A command registered through `Register` SHALL be available in every session in the process, its declared argument count SHALL be enforced before its `Run`, its error SHALL reach the caller of `Execute`, and the session SHALL save its variables after it succeeds. `Register` SHALL refuse a name already registered, a built-in's included, and a handler without `Run`. The same names in `cli/commands` SHALL carry a `Deprecated:` notice pointing at `engine/dsl`.

#### Scenario: A registered command in a session
- **WHEN** a program registers `zzdouble` through `engine/dsl`, opens a session and runs `zzdouble x as y`
- **THEN** the command's output reaches the session's output, and `y` is saved in the environment

#### Scenario: An argument past the declared count
- **WHEN** `zzdouble`, declared with `MaxArgs(1).WithAlias()`, is run as `zzdouble x y z`
- **THEN** `Execute` returns an error and `Run` is not called

#### Scenario: A taken name
- **WHEN** a program registers a command named `newdl`
- **THEN** `Register` returns an error
