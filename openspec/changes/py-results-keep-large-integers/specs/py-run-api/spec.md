## ADDED Requirements

### Requirement: An integer in a result keeps every digit

The runners SHALL decode an integer in the value Python passes to `insyra.Return` without changing it. An integer above 2^53 in magnitude SHALL come back as an `int64`, or as a `uint64` when it is above the `int64` range, wherever the result type holds an `any`, a table cell or a list cell. An integer within ±2^53 and a float SHALL come back as a float64 there, as before. Bound into an integer field, an integer SHALL be exact whenever the field's type holds it. An integer beyond 64 bits SHALL come back as the nearest float64. `insyra.Return` SHALL convert each column of a pandas or polars DataFrame on its own, so an integer column beside a float one is sent as integers.

#### Scenario: A large id into an int64
- **WHEN** Python returns `2**53 + 1` and the result type is `int64`
- **THEN** the result is 9007199254740993

#### Scenario: A large id into any
- **WHEN** Python returns `{"id": 2**53 + 1, "n": 3}` and the result type is `map[string]any`
- **THEN** `id` is the `int64` 9007199254740993 and `n` is the float64 3

#### Scenario: A DataFrame of ids
- **WHEN** Python returns a DataFrame whose column `id` holds `2**53 + 1` and `2**63 - 1`, and the result type is `*insyra.DataTable`
- **THEN** the cells are the `int64` values 9007199254740993 and 9223372036854775807

#### Scenario: A DataFrame of ids and floats
- **WHEN** Python returns a pandas or polars DataFrame with an integer column `id` holding `2**53 + 1` beside a float column `x`
- **THEN** the `id` cell is the `int64` 9007199254740993 and the `x` cells are float64

#### Scenario: Above the int64 range
- **WHEN** Python returns `2**64 - 1` and the result type is `any`
- **THEN** the result is the `uint64` 18446744073709551615

## MODIFIED Requirements

### Requirement: Tables and lists inside a result are decoded as tables and lists

When the type a result is decoded into holds `*insyra.DataTable`, `*insyra.DataList`, `insyra.IDataTable` or `insyra.IDataList` below its top level, in a struct field, a map value, a slice or array element, or behind a pointer, `RunCode`, `Run` and the other runners SHALL decode each such part with the table or list decoder, and every other part as `encoding/json` decodes it: struct fields matched and hidden by its rules, including its rules for embedded structs, `-` and `,string`; map keys of string kind, integer kind or with `UnmarshalText`; a value that already holds something keeping what the result leaves out; and `None` setting a pointer, slice, map or interface to nil and leaving a struct as it is. A type that holds an `any` SHALL be decoded the same way: an `any` SHALL take the decoded value, or, when it already holds a non-nil pointer and the value is not `None`, SHALL be decoded into what the pointer points to, as in `encoding/json`. An integer field SHALL take a number only when its type holds it; a number it cannot hold, or one with a fraction, SHALL be an error. A type that implements `json.Unmarshaler` or `encoding.TextUnmarshaler`, or that holds no table, list or `any`, SHALL be decoded through JSON as before, unless the result holds a number JSON would wrap around in an integer field. An embedded `*insyra.DataTable` or `*insyra.DataList` SHALL follow the rule for an embedded struct: with a tag it is a field under that name, and without one it is decoded from the whole value, at any depth of the result, and a value it cannot decode SHALL be an error. Beside other fields it SHALL take the whole value only when that value is a table or list as `insyra.Return` marks one, or an array; an object without that mark SHALL be decoded into the other fields. An embedded `insyra.IDataTable` or `insyra.IDataList` SHALL be a field named after its type, as an embedded interface is in `encoding/json`. Decoding SHALL never end the program, whatever the result type.

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

#### Scenario: An any holding a pointer
- **WHEN** the result type is a struct whose `any` field already holds a pointer to a struct, and Python returns an object for that field
- **THEN** the object is decoded into the struct the pointer points to, and the field still holds the pointer

#### Scenario: An integer the field cannot hold
- **WHEN** Python returns `2**63` and the result type is `int64`, or `2**64` and it is `uint64`
- **THEN** the call returns an error naming the type, instead of a number wrapped around
