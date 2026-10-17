# data-type Specification

## Purpose
Lets a program ask what kind of values a column holds (numbers, text, booleans, times, something else, a mix, or nothing) before it analyses the column, with one rule the whole library can share instead of each part judging it privately.

## Requirements

### Requirement: A column reports the kind of values it holds

`DataList.DataType()` SHALL return the one data type all of the list's non-missing values share, `DataTypeMixed` when they do not share one, and `DataTypeEmpty` when there are none. `nil` and `NaN` SHALL be treated as missing. Every integer, unsigned and float width, a named type over one, and a decimal SHALL count as `DataTypeNumber`, and a string SHALL count as `DataTypeString` whatever its content. `DataTable.ColDataTypes()` SHALL return each column's data type in column order.

#### Scenario: Numbers with gaps

- **WHEN** a list holds `10`, `2.5`, `nil` and `NaN`
- **THEN** its data type is `DataTypeNumber`

#### Scenario: Text that looks numeric

- **WHEN** a list holds `"0050"` and `"2330"`
- **THEN** its data type is `DataTypeString`

#### Scenario: Different kinds together

- **WHEN** a list holds `1` and `"a"`
- **THEN** its data type is `DataTypeMixed`

#### Scenario: The type view states the verdict

- **WHEN** `ShowTypes` prints a table whose columns are a number, a text and a mixed column
- **THEN** its first row, `DataType`, reads `number`, `string` and `mixed` above the per-cell Go types

#### Scenario: Only missing values

- **WHEN** a list holds only `nil` and `NaN`
- **THEN** its data type is `DataTypeEmpty`

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
