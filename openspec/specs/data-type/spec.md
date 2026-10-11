# data-type Specification

## Purpose
What `DataList.DataType` and `DataTable.ColDataTypes` report about the kind of values a column holds, which values count as missing, and how `ShowTypes` shows it.

## Requirements

### Requirement: A column reports the kind of values it holds

`DataList.DataType()` SHALL return the one data type all of the list's non-missing values share, `DataTypeMixed` when they do not share one, and `DataTypeEmpty` when there are none.

#### Scenario: Numbers with gaps

- **WHEN** a list holds `10`, `2.5`, `nil` and `NaN`
- **THEN** its data type is `DataTypeNumber`

#### Scenario: Text that looks numeric

- **WHEN** a list holds `"0050"` and `"2330"`
- **THEN** its data type is `DataTypeString`

#### Scenario: Different kinds together

- **WHEN** a list holds `1` and `"a"`
- **THEN** its data type is `DataTypeMixed`

#### Scenario: Decimal is DataTypeOther

- **WHEN** a list holds a decimal value
- **THEN** its data type is `DataTypeOther`

#### Scenario: NaN of any float kind is missing

- **WHEN** a list holds only `float32(math.NaN())` or a named float NaN
- **THEN** its data type is `DataTypeEmpty`

#### Scenario: Only missing values

- **WHEN** a list holds only `nil` and `NaN`
- **THEN** its data type is `DataTypeEmpty`

### Requirement: Missing values and types are classified consistently

For data type determination, `nil` and `NaN` of any float kind SHALL be treated as missing, every integer, unsigned, and float width (and named types over them) SHALL count as `DataTypeNumber`, decimals SHALL count as `DataTypeOther`, and strings SHALL count as `DataTypeString`.

#### Scenario: Custom float NaN treated as missing

- **WHEN** a list contains custom named float values that are NaN
- **THEN** those values are treated as missing rather than contributing to DataTypeNumber

#### Scenario: String content does not affect classification

- **WHEN** a list holds string representations of booleans or numbers
- **THEN** its data type is `DataTypeString`

### Requirement: DataTable reports column data types in order

`DataTable.ColDataTypes()` SHALL return each column's data type in column order.

#### Scenario: Table column types in order

- **WHEN** `ColDataTypes()` is called on a table with a number column followed by a string column
- **THEN** the returned slice contains `DataTypeNumber` followed by `DataTypeString`

### Requirement: ShowTypes displays the data type verdict

`ShowTypes` SHALL print a `DataType` row (or `DataType:` line) whose column width fits the whole word.

#### Scenario: The type view states the verdict and fits the word

- **WHEN** `ShowTypes` prints a table whose columns are a number, a text and a mixed column
- **THEN** its first row, `DataType`, reads `number`, `string` and `mixed` above the per-cell Go types, and columns are wide enough to show the full word
