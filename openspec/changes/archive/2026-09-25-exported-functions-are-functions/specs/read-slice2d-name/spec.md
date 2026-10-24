## ADDED Requirements

### Requirement: A 2D slice is read by one name

`ReadSlice2D` SHALL be the function that converts a 2D slice into a DataTable. `Slice2DToDataTable` SHALL remain until 0.4, marked Deprecated with a notice that names `ReadSlice2D` and says it is removed in 0.4, and SHALL return exactly what `ReadSlice2D` returns.

#### Scenario: The deprecated name is used
- **WHEN** the same slice is passed to `Slice2DToDataTable` and to `ReadSlice2D`
- **THEN** both return tables with the same shape and values

#### Scenario: The notice
- **WHEN** the doc comment of `Slice2DToDataTable` is read
- **THEN** it has a `Deprecated:` paragraph naming `ReadSlice2D` and saying it is removed in 0.4
