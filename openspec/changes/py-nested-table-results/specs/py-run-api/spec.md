## ADDED Requirements

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
