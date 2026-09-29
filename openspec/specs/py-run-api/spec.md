# py-run-api Specification

## Purpose
How Go code runs Python through `py`: the typed `Run`, the context forms of the runners and of the `uv pip` commands, what a nil, finished or ending context does to a call, and what a failed Python process returns.
## Requirements
### Requirement: Run returns the result as the type asked for

`py.Run[T any](ctx context.Context, code string, args ...any) (T, error)` SHALL run `code` with `$v1`, `$v2`, … replaced from `args` as `RunCodef` replaces them, and SHALL return the value Python passed to `insyra.Return` decoded into a `T` by the rules `RunCode` binds with. When the run or the decoding fails, it SHALL return the zero value of `T` and the error. When `T` is `insyra.DataTable` or `insyra.DataList` itself, `Run` SHALL return an error pointing to the pointer type before it starts anything.

#### Scenario: A struct
- **WHEN** Python returns `{"name": "insyra", "count": 3}` and `T` is a struct with fields tagged `name` and `count`
- **THEN** `Run` returns that struct filled in and a nil error

#### Scenario: A DataFrame
- **WHEN** Python returns a pandas DataFrame and `T` is `*insyra.DataTable`
- **THEN** `Run` returns a table with the DataFrame's rows, columns and column names

#### Scenario: A value table
- **WHEN** `Run[insyra.DataTable]` is called
- **THEN** it returns an error naming `Run[*insyra.DataTable]` without starting Python

#### Scenario: Python raises
- **WHEN** the code raises an exception
- **THEN** `Run` returns the zero value of `T` and an error carrying the exception's message

#### Scenario: Placeholders
- **WHEN** `Run` is called with `code` `x = $v1` and the argument `[]int{1, 2}`
- **THEN** the script Python runs contains `x = [1, 2]`

### Requirement: The context forms handle a nil or finished context

`Run`, `RunCodeContext`, `RunCodefContext`, `RunFileContext`, `RunFilefContext`, `PipInstallContext` and `PipUninstallContext` SHALL return an error for a nil context instead of panicking, and SHALL return `ctx.Err()` without preparing the environment or starting a process when the context is already done. When the context ends while Python or `uv pip` runs, the process SHALL be stopped and the call SHALL return `ctx.Err()`.

#### Scenario: Nil context
- **WHEN** any of these functions is called with a nil context
- **THEN** it returns an error that says the context is nil, and the program does not panic

#### Scenario: Context already cancelled
- **WHEN** any of these functions is called with a cancelled context
- **THEN** it returns an error for which `errors.Is(err, context.Canceled)` holds, at once

#### Scenario: Deadline during the run
- **WHEN** the context's deadline passes while Python or `uv pip install` runs
- **THEN** the call returns promptly with an error for which `errors.Is(err, context.DeadlineExceeded)` holds

### Requirement: A failed Python process is the call's error

When the Python process exits with an error before it returns a value, whether it crashed or was killed because the context ended, the call SHALL return an error: `ctx.Err()` when the context ended, the process's error otherwise. It SHALL NOT report the run as successful with no result. A result Python delivered before the process failed SHALL be returned.

#### Scenario: The process fails and exits at once
- **WHEN** the process has reported its error and the process is done, both before the runner looks
- **THEN** the runner returns the error, every time

#### Scenario: The process fails after delivering its result
- **WHEN** Python has delivered a result and the process then reports an error
- **THEN** the runner returns the delivered result

### Requirement: PipInstall and PipUninstall have context forms

`PipInstallContext(ctx, dep)` and `PipUninstallContext(ctx, dep)` SHALL do what `PipInstall` and `PipUninstall` do, with the context bounding the environment setup and the `uv pip` command. `PipInstall(dep)` and `PipUninstall(dep)` SHALL call them with `context.Background()`, and a name starting with `-` SHALL still be refused before anything runs.

#### Scenario: Refused name
- **WHEN** `PipInstallContext(ctx, "--requirement=/etc/passwd")` is called
- **THEN** it returns an error naming `PipInstall` before the environment is prepared

### Requirement: RunCodeWithTimeout is deprecated

`RunCodeWithTimeout` SHALL keep its meaning, running the code under a context from `context.WithTimeout`, and its doc comment SHALL say `Deprecated: use RunCodeContext` with `context.WithTimeout` and that it is removed in the release after the one that deprecated it.

#### Scenario: Timeout
- **WHEN** `RunCodeWithTimeout(300*time.Millisecond, nil, code)` runs code that does not finish
- **THEN** it returns an error for which `errors.Is(err, context.DeadlineExceeded)` holds

