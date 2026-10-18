## MODIFIED Requirements

### Requirement: ProcessData reports failure as an error

`ProcessData` SHALL return `([]any, error)`. It SHALL read a slice or an array element by element, an `IDataList` by its values, and a pointer to a slice or an array by what it points to. It SHALL return an error and a nil slice for any other type, for a nil input and for a nil pointer, a nil `*DataList` included, and SHALL NOT panic. An empty slice SHALL give an empty, non-nil slice and a nil error.

#### Scenario: A value it cannot read

- **WHEN** a caller passes `42`, `nil` or a nil `*DataList`
- **THEN** `ProcessData` returns a nil slice and a non-nil error

#### Scenario: The functions that read weights and samples through it

- **WHEN** `stats.Skewness` is called on a nil `*DataList`
- **THEN** it returns an error prefixed `sample:`, neither crashing nor reporting `empty data`
- **AND** `WeightedMean` no longer reads its weights through `ProcessData`: it takes a `[]float64`, so weights it could not read do not compile

## REMOVED Requirements

### Requirement: The big.Rat helpers never panic
**Reason**: `SqrtRat`, `PowRat`, `SortTimes` and `F64orRat` were removed in 0.4, the release after the one that deprecated them.
**Migration**: Use `math/big` directly, and `slices.SortFunc` for sorting times.

### Requirement: A 2D slice is read by one name
**Reason**: It named deprecated APIs that were removed in 0.4, and its scenarios cannot be edited in place; it is restated under a new name in this change.
**Migration**: See the requirement "A 2D slice is read by ReadSlice2D".

## ADDED Requirements

### Requirement: A 2D slice is read by ReadSlice2D

`ReadSlice2D` SHALL be the function that converts a 2D slice into a DataTable. `Slice2DToDataTable` SHALL NOT exist.

#### Scenario: A 2D slice is read
- **WHEN** a caller passes a 2D slice to `ReadSlice2D`
- **THEN** it returns a table with the slice's shape and values
