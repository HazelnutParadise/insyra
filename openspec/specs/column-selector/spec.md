# column-selector Specification

## Purpose
整個函式庫只用一套方式指定欄位：裸字串是 Excel 式索引，`Name(...)` 是欄名，`int` 是位置。欄名只影響錯誤訊息、不參與解析，所以同一個呼叫在任何資料上都指向同一欄；核心存取器另外保留 `ByIndex`／`ByName`／`ByNumber` 的明講寫法，分析操作則靠選擇器本身表達。
## Requirements
### Requirement: One selector says which column

Every parameter and every config field that chooses a column SHALL accept the same three forms and no others: a `string` SHALL be an Excel-style column index (`"A"`, `"B"`, ... `"AA"`, case-insensitive), a `Name` SHALL be a column name compared exactly, and an `int` SHALL be a 0-based position counting from the end when negative. A string SHALL NOT be looked up as a column name under any circumstance, and a name SHALL NOT be read as an index.

#### Scenario: The same string means the same thing everywhere

- **WHEN** `"B"` is given to a core accessor and to an analysis operation on the same table
- **THEN** both address the second column

#### Scenario: A name is said, not guessed

- **WHEN** a column is addressed as `Name("price")`
- **THEN** the column named `price` is used, and no Excel-style reading is attempted

#### Scenario: A selector of another type is refused

- **WHEN** a selector is neither a string, a `Name`, nor an int
- **THEN** the call fails naming the type it was given and the three forms it accepts

### Requirement: A name that could be read as an index is reported, not resolved

When a bare string resolves to an in-range index and the table also has a column whose name is exactly that string, the call SHALL use the index and SHALL record a warning naming both readings. The presence of such a column SHALL NOT change which column is used.

#### Scenario: A column named like a letter

- **WHEN** a table whose first column is named `B` is asked for `"B"`
- **THEN** the second column is returned and a warning names both readings

#### Scenario: A name where an index goes

- **WHEN** a table with a column named `price` is asked for `"price"`
- **THEN** the call fails, because `price` decodes to an index past the last column, and the failure says to write `Name("price")`

### Requirement: Every core accessor keeps an explicit spelling

`GetCol`, `UpdateCol`, `SetColToRowNames` and the four `Replace*InCol` methods SHALL each have `ByIndex`, `ByName` and `ByNumber` counterparts where that kind is meaningful, so a caller can be explicit without building a selector. The analysis operations SHALL NOT grow such counterparts, because their parameters already take the selector.

#### Scenario: The explicit and the generic agree

- **WHEN** `GetCol(Name("price"))` and `GetColByName("price")` run on the same table
- **THEN** both return the same column

#### Scenario: An explicit method refuses the other kind

- **WHEN** `GetColByIndex("price")` is called and `price` decodes past the last column
- **THEN** it fails naming the index it decoded, without trying the name

### Requirement: A fitter's failure on a column says why

The scalers, the simple imputer and the one-hot, label and ordinal encoders SHALL report a column they cannot resolve with the same explanation every other selector gives, including the `Name(...)` form when the table has a column of that name.

#### Scenario: A scaler given a name as a bare string

- **WHEN** `NewStandardScaler().FitTransform(dt, "Age")` runs on a table with a column named `Age`
- **THEN** it fails, and the error says to write `Name("Age")`

#### Scenario: An encoder given a name as a bare string

- **WHEN** `LabelEncode` is given `Column: "segment"` on a table with a column named `segment`
- **THEN** it fails, and the error says to write `Name("segment")`

