## ADDED Requirements

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
