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

### Requirement: Tables and lists inside a result are decoded as tables and lists

When the type a result is decoded into holds `*insyra.DataTable`, `*insyra.DataList`, `insyra.IDataTable` or `insyra.IDataList` below its top level, in a struct field, a map value, a slice or array element, or behind a pointer, `RunCode`, `Run` and the other runners SHALL decode each such part with the table or list decoder, and every other part as `encoding/json` decodes it: struct fields matched and hidden by its rules, including its rules for embedded structs, `-` and `,string`; map keys of string kind, integer kind or with `UnmarshalText`; a value that already holds something keeping what the result leaves out; and `None` setting a pointer, slice, map or interface to nil and leaving a struct as it is. A type that implements `json.Unmarshaler` or `encoding.TextUnmarshaler`, or that holds no table or list, SHALL be decoded through JSON as before. An embedded `*insyra.DataTable` or `*insyra.DataList` SHALL follow the rule for an embedded struct: with a tag it is a field under that name, and without one it is decoded from the whole value, at any depth of the result, and a value it cannot decode SHALL be an error. Beside other fields it SHALL take the whole value only when that value is a table or list as `insyra.Return` marks one, or an array; an object without that mark SHALL be decoded into the other fields. An embedded `insyra.IDataTable` or `insyra.IDataList` SHALL be a field named after its type, as an embedded interface is in `encoding/json`. Decoding SHALL never end the program, whatever the result type.

`insyra.Return` SHALL turn a pandas or polars DataFrame or Series anywhere inside a dict, list or tuple into the payload it sends for one at the top level.

#### Scenario: A struct holding a table
- **WHEN** Python returns `{"table": <a DataFrame with columns a and b>, "score": 7}` and the result type is a struct with a `*insyra.DataTable` field tagged `table` and a `float64` field tagged `score`
- **THEN** the table field holds the DataFrame's rows and column names and the score field is 7

#### Scenario: A field named DataTable
- **WHEN** the struct's table field is named `DataTable` but is not embedded
- **THEN** it is decoded as that field, and the struct's other fields are decoded too

#### Scenario: Tables in a map and lists in a slice
- **WHEN** the result type is `map[string]*insyra.DataTable` or `[]*insyra.DataList`
- **THEN** each value or element is decoded as a table or a list

#### Scenario: An embedded wrapper
- **WHEN** the result type is a struct that embeds `*insyra.DataTable` without a tag and Python returns a DataFrame
- **THEN** the embedded field holds the table, as before, also when the struct sits in a slice or a map

#### Scenario: An isr wrapper given the other kind of result
- **WHEN** a DataFrame is decoded into an `isr` list
- **THEN** the call returns an error and the list is left as it was

#### Scenario: A tagged embedded table beside other fields
- **WHEN** the struct embeds `*insyra.DataTable` tagged `table` beside a field tagged `score`, and Python returns `{"table": <a DataFrame>, "score": 7}`
- **THEN** the table comes from the key `table` and the score is 7

#### Scenario: An untagged embedded table beside other fields
- **WHEN** the struct embeds `*insyra.DataTable` without a tag beside a field `Score`, and the result is the object `{"a": 1, "b": 2, "Score": 7}`
- **THEN** the table is left as it was and `Score` is 7; for a DataFrame instead, the table holds it

#### Scenario: Fields matched as encoding/json matches them
- **WHEN** a struct holding a table is decoded from keys that differ only in case, keys for a field hidden by another or for two fields of one name at one depth, or a field tagged `,string`
- **THEN** every field holds what `encoding/json` gives for the same keys

#### Scenario: A struct that embeds itself
- **WHEN** the result type is a struct that embeds a pointer to itself, or two structs embed pointers to each other
- **THEN** the result decodes, and no embedded pointer is allocated unless a key reaches it

#### Scenario: A dict holding a DataFrame from Python
- **WHEN** the code runs `insyra.Return({"table": df, "score": 7})` in the real environment
- **THEN** the call succeeds and the table arrives as a table

### Requirement: An empty DataFrame keeps its column names

A DataFrame with no rows SHALL be decoded into a table with its columns and their names, not renamed with numeric suffixes.

#### Scenario: A filter that matched nothing
- **WHEN** Python returns a DataFrame with columns `a` and `b` and no rows
- **THEN** the table has 0 rows and 2 columns named `a` and `b`

### Requirement: Placeholders are replaced in one pass with Python literals

`RunCodef`, `RunFilef`, `RunCodefContext`, `RunFilefContext` and `Run` SHALL find the placeholders `$v1`, `$v2`, … in one pass over the template, each read by its whole number, and SHALL replace each with the Python literal of the argument at that position. Text inserted for one placeholder SHALL never be searched for another. A placeholder whose number is past the last argument, or written with a leading zero, SHALL be left as written. Only the arguments the template uses SHALL be converted. An argument that cannot be written as a Python literal SHALL make the call return an error naming its placeholder before Python starts, and SHALL never be written into the script as formatted text.

#### Scenario: A value holding a placeholder
- **WHEN** the template is `title = $v1` and `label = $v2` on two lines, the first argument is the text `$v2` and the second is `+__import__('os').system('id')+`
- **THEN** the script holds `title = "$v2"` and `label = "+__import__('os').system('id')+"`

#### Scenario: Ten or more arguments
- **WHEN** the template is `x = $v10 + $v1` and the arguments are the texts `a` to `j`
- **THEN** the script holds `x = "j" + "a"`

#### Scenario: A value Python cannot read
- **WHEN** the template uses `$v1` and the argument is `[]any{"__import__('os').system('id'),", math.NaN()}`
- **THEN** the call returns an error naming `$v1` and Python does not start

#### Scenario: A non-finite float in a slice
- **WHEN** the template uses `$v1` and the argument is `[]float64{1, math.NaN()}`, or holds an infinity
- **THEN** the call returns an error naming `$v1`, as for a single `math.NaN()`

#### Scenario: An argument the template does not use
- **WHEN** the template is `x = $v2` and the arguments are `math.NaN()` and `3`
- **THEN** the script holds `x = 3` and the call does not fail

### Requirement: A NaN or an infinity in a result comes back as one

When the value Python passes to `insyra.Return` holds a NaN, an `inf` or a `-inf`, the runners SHALL decode it as `math.NaN()`, `math.Inf(1)` or `math.Inf(-1)` wherever the result type can hold a float64: a table or list cell, a float field, a slice, array or map element, or an `any`. Python's `None` SHALL still come back as `nil`. Decoding a NaN or an infinity into a type that cannot hold one SHALL be an error. Text inside strings SHALL be left as it is, and a result holding no NaN or infinity SHALL decode exactly as before.

#### Scenario: A DataFrame with a missing value
- **WHEN** Python returns a DataFrame whose column `a` holds `1.0` and `NaN`, and the result type is `*insyra.DataTable`
- **THEN** the table's second cell in `a` is a NaN and the call returns no error

#### Scenario: Floats outside tables
- **WHEN** Python returns `{"x": [1.5, nan, inf, -inf]}` and the result type is `map[string][]float64`, or `any`
- **THEN** the values are 1.5, a NaN, `+Inf` and `-Inf`

#### Scenario: A NaN for an int
- **WHEN** Python returns a NaN and the result type is `int`
- **THEN** the call returns an error naming the NaN

#### Scenario: The names inside text
- **WHEN** Python returns the string `NaN Infinity`
- **THEN** the string comes back unchanged

### Requirement: A result the Go side cannot read is an error

When the Go side cannot decode the message `insyra.Return` sends, it SHALL answer with an error, and unless the run then delivers another result, the call SHALL return an error saying why rather than `nil`. `insyra.Return` SHALL raise when the answer is an error or when the connection closes without an answer.

#### Scenario: An integer too large for a float64
- **WHEN** Python returns `10**400`
- **THEN** the call returns an error saying the number does not fit in a float64

#### Scenario: Another result after the refusal
- **WHEN** the code catches the exception `insyra.Return(10**400)` raises and returns `"fallback"`
- **THEN** the call returns `"fallback"` and no error

#### Scenario: No answer
- **WHEN** the Go side closes the connection without answering
- **THEN** `insyra.Return` raises in Python, and the process exits with an error

