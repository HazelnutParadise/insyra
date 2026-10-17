## ADDED Requirements

### Requirement: Run decodes the result into the type asked for

`py.Run[T]` SHALL run the code and return what it passed to `insyra.Return` decoded into a `T`. On any failure it SHALL return the zero `T` and the error.

#### Scenario: A typed result
- **WHEN** the code returns a list of dictionaries and `T` is `[]map[string]int`
- **THEN** the slice is returned with a nil error

#### Scenario: Failure
- **WHEN** the code exits with an error
- **THEN** the zero `T` and the error are returned

### Requirement: Context forms refuse a nil or ended context

Every `py` function that takes a context SHALL return an error for a nil context and `ctx.Err()` for one that has ended, before starting a process. `PipInstallContext` and `PipUninstallContext` SHALL stop the process when the context ends.

#### Scenario: Nil context
- **WHEN** a context form is called with a nil context
- **THEN** it returns an error and does not panic

#### Scenario: Cancelled install
- **WHEN** the context ends while a pip call runs
- **THEN** the call returns the context's error

### Requirement: RunCodeWithTimeout is deprecated

`RunCodeWithTimeout` SHALL keep its behaviour and carry a Deprecated notice naming `RunCodeContext` and the removal in 0.4.

#### Scenario: Timeout still applies
- **WHEN** the code outlasts the timeout
- **THEN** the error matches `context.DeadlineExceeded`
