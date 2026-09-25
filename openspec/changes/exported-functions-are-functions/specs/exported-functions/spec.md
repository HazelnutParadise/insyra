## ADDED Requirements

### Requirement: An exported function is a function declaration

No exported package-level variable in the module SHALL hold a function literal or a function declared in the module. Each such function SHALL be exported with `func`, so a caller cannot replace it and the documentation lists it among functions.

#### Scenario: The module is scanned
- **WHEN** the test parses every non-test Go file in the module
- **THEN** it finds no exported variable whose value is a function literal or names a function declared in the module

#### Scenario: A conversion is called
- **WHEN** a caller writes `insyra.ToFloat64Safe(v)`
- **THEN** it compiles and behaves as before, and `insyra.ToFloat64Safe = f` does not compile

### Requirement: A 2D slice is read by one name

`ReadSlice2D` SHALL be the function that converts a 2D slice into a DataTable. `Slice2DToDataTable` SHALL remain for one release, marked Deprecated, and return exactly what `ReadSlice2D` returns.

#### Scenario: The deprecated name is used
- **WHEN** a caller passes the same slice to `Slice2DToDataTable` and to `ReadSlice2D`
- **THEN** both return tables with the same shape and values
