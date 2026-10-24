# py-run-api Specification

## Purpose
How Go code runs Python through `py` and gets its result back: how a result is decoded into the caller's Go value, including tables and lists nested inside it, NaN and infinities, and results the Go side cannot read, and how placeholders in the code are filled in.

## Requirements

### Requirement: Tables and lists below the top level are decoded part by part

The runners SHALL decode a result into the caller's type as `encoding/json` does, except that a part whose type holds `*insyra.DataTable`, `*insyra.DataList`, `insyra.IDataTable` or `insyra.IDataList` SHALL be decoded by the table or list decoder, wherever it sits. A type that implements `json.Unmarshaler` or `encoding.TextUnmarshaler`, or holds no table or list, SHALL be decoded through JSON. Decoding SHALL never end the program.

#### Scenario: A struct holding a table
- **WHEN** Python returns `{"table": <a DataFrame with columns a and b>, "score": 7}` and the result type is a struct with a `*insyra.DataTable` field tagged `table` and a `float64` field tagged `score`
- **THEN** the table field holds the DataFrame's rows and column names and the score field is 7

#### Scenario: Tables in a map and lists in a slice
- **WHEN** the result type is `map[string]*insyra.DataTable` or `[]*insyra.DataList`
- **THEN** each value or element is decoded as a table or a list

#### Scenario: A struct without a table
- **WHEN** the result type holds no table or list
- **THEN** every field is decoded through JSON as before

#### Scenario: A dict holding a DataFrame from Python
- **WHEN** the code runs `insyra.Return({"table": df, "score": 7})`, which sends the DataFrame inside the dict as its payload, in the real environment
- **THEN** the call succeeds and the table arrives as a table

### Requirement: A field named DataTable or DataList takes a whole result

When the result is itself a table or a list, a struct field named `DataTable` or `DataList`, embedded or not and at any depth, SHALL receive the whole result when its type is the matching pointer type and its `json` tag does not send it elsewhere. A result that is an object SHALL be decoded key by key into the struct's fields instead.

#### Scenario: A whole table into a named field
- **WHEN** Python returns a DataFrame and the result type is a struct with a non-embedded `*insyra.DataTable` field named `DataTable`
- **THEN** the field holds the whole table

#### Scenario: A dict beside a named field
- **WHEN** Python returns `{"Score": 7, "Table": <a DataFrame>}` and the result type is a struct with `Score float64` and `Table *insyra.DataTable`
- **THEN** `Score` is 7 and `Table` holds the table

### Requirement: An embedded table or list follows the rule for embedded structs

An embedded `*insyra.DataTable` or `*insyra.DataList` SHALL follow the rule for an embedded struct: with a tag it SHALL be a field under that name, and without one it SHALL be decoded from the whole value. Beside other fields an untagged one SHALL take the whole value only when that value is a table or list as `insyra.Return` marks one, or an array; an object without that mark goes to the other fields. An embedded `insyra.IDataTable` or `insyra.IDataList` SHALL be a field named for its type.

#### Scenario: An embedded wrapper
- **WHEN** the result type is a struct that embeds `*insyra.DataTable` without a tag and Python returns a DataFrame
- **THEN** the embedded field holds the table, as before, also when the struct sits in a slice or a map

#### Scenario: A tagged embedded table beside other fields
- **WHEN** the struct embeds `*insyra.DataTable` tagged `table` beside a field tagged `score`, and Python returns `{"table": <a DataFrame>, "score": 7}`
- **THEN** the table comes from the key `table` and the score is 7

#### Scenario: An untagged embedded table beside other fields
- **WHEN** the struct embeds `*insyra.DataTable` without a tag beside a field `Score`, and the result is the object `{"a": 1, "b": 2, "Score": 7}`
- **THEN** the table is left as it was and `Score` is 7; for a DataFrame instead, the table holds it

#### Scenario: An embedded table given the other kind of result
- **WHEN** a DataFrame is decoded into an `isr` list
- **THEN** the call returns an error and the list is left as it was

### Requirement: Field matching and existing values follow encoding/json

Struct fields SHALL be matched and hidden by `encoding/json`'s rules, including its rules for embedded structs, `-`, `,string` and two fields of one name at one depth; a key SHALL be matched with Unicode simple folding, as `encoding/json` folds it. A map key SHALL be of string or integer kind or implement `UnmarshalText`. A value that already holds something SHALL keep what the result leaves out. `None` SHALL set a pointer, slice, map or interface to nil and SHALL leave a struct as it is.

#### Scenario: Fields matched as encoding/json matches them
- **WHEN** a struct holding a table is decoded from keys that differ only in case, a key for a field hidden by another or for two fields of one name at one depth, or a field tagged `,string`
- **THEN** every field holds what `encoding/json` gives for the same keys

#### Scenario: A key folded like encoding/json folds it
- **WHEN** a key is the dotless `ı` and a field is named `I`
- **THEN** the field holds what `encoding/json` gives for that key

#### Scenario: A struct that embeds itself
- **WHEN** the result type is a struct that embeds a pointer to itself, or two structs embed pointers to each other
- **THEN** the result decodes, and no embedded pointer is allocated unless a key reaches it

#### Scenario: Tables in a map with integer or text keys
- **WHEN** the result type is `map[int]*insyra.DataTable` or a map whose key implements `UnmarshalText`
- **THEN** each value is decoded as a table

#### Scenario: Decoding into an existing value
- **WHEN** the destination already holds a value and the result leaves a field out
- **THEN** that field keeps what it held

#### Scenario: None into a wrapper
- **WHEN** the result is `None` and the destination is a pointer, slice, map or interface, or a struct
- **THEN** the pointer, slice, map or interface becomes nil and the struct is left as it is

### Requirement: An empty DataFrame keeps its column names

A DataFrame with no rows SHALL be decoded into a table with its columns and their names, not renamed with numeric suffixes.

#### Scenario: A filter that matched nothing
- **WHEN** Python returns a DataFrame with columns `a` and `b` and no rows
- **THEN** the table has 0 rows and 2 columns named `a` and `b`

### Requirement: A NaN or an infinity in a result comes back as one

When the value Python passes to `insyra.Return` holds a NaN, an `inf` or a `-inf`, the runners SHALL decode it as `math.NaN()`, `math.Inf(1)` or `math.Inf(-1)` wherever the result type can hold a float64: a table or list cell, a float field, a slice, array or map element, or an `any`. Python's `None` SHALL still come back as `nil`. Decoding a NaN or an infinity into a type that cannot hold one SHALL be an error, and a result holding none SHALL decode exactly as before.

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

### Requirement: A pandas DataFrame's name is never one of its columns

When `insyra.Return` sends a pandas DataFrame, the table SHALL take a `name` as its name only from the frame itself: a column called `name` SHALL NOT be read as the name, while a name set on the frame, which pandas keeps in the instance's `__dict__`, SHALL be.

#### Scenario: A column called name
- **WHEN** Python returns `pd.DataFrame({"name": ["x", "y"], "v": [1, 2]})`
- **THEN** the table has no name and its columns are `name` and `v`

#### Scenario: A name set on the DataFrame
- **WHEN** Python sets `df.name = "scores"` on a DataFrame without a column called `name` and returns it
- **THEN** the table is named `scores`

#### Scenario: A name column beside an explicit name
- **WHEN** a DataFrame has a column called `name` and a name set on the frame itself
- **THEN** the table is named that name

### Requirement: Placeholders are replaced in one pass with Python literals

`RunCodef`, `RunFilef` and their `Context` forms SHALL find the placeholders `$v1`, `$v2`, … in one pass, each by its whole number, replacing each with the Python literal of that argument. Text inserted for one placeholder is never searched for another. A placeholder past the last argument, or with a leading zero, is left as written. An argument that cannot be written as a Python literal SHALL make the call return an error naming its placeholder before Python starts, never as formatted text.

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

#### Scenario: A placeholder with no argument
- **WHEN** the template uses `$v3` with two arguments, or writes `$v01`
- **THEN** the placeholder is left in the script as written

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
