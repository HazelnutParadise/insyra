## ADDED Requirements

### Requirement: A NaN of any floating-point type counts as missing

`DataList.DataType()` SHALL treat a `NaN` of any floating-point type, including `float32` and a named type whose underlying type is a float, as missing, the same as a `float64` `NaN`.

#### Scenario: A float32 NaN among numbers

- **WHEN** a list holds `10` and a `float32` NaN
- **THEN** its data type is `DataTypeNumber`

#### Scenario: Only a named float type's NaN

- **WHEN** a list holds only a NaN of a named type over `float64`
- **THEN** its data type is `DataTypeEmpty`

### Requirement: The type view keeps the whole DataType word

`ShowTypes` SHALL size each column so that its `DataType` word is printed whole, however narrow the column's values are, and MUST NOT shorten the word to fit.

#### Scenario: A narrow number column beside an empty one

- **WHEN** `ShowTypes` prints a table with a column holding `1` and a column holding only `nil`, with colour off
- **THEN** the output contains `number` and `empty` in full
