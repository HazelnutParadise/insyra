# exported-functions Specification

## Purpose
How Insyra exports its functions. A function a caller can reach is declared with `func`, so nobody can swap out the implementation the library itself calls, and it has one name, so a reader never wonders whether two spellings differ.

## Requirements

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

### Requirement: ProcessData reports failure as an error

`ProcessData` SHALL return `([]any, error)`. It SHALL read a slice or an array element by element, an `IDataList` by its values, and a pointer to a slice or an array by what it points to. It SHALL return an error and a nil slice for any other type, for a nil input and for a nil pointer, a nil `*DataList` included, and SHALL NOT panic. An empty slice SHALL give an empty, non-nil slice and a nil error.

#### Scenario: A value it cannot read

- **WHEN** a caller passes `42`, `nil` or a nil `*DataList`
- **THEN** `ProcessData` returns a nil slice and a non-nil error

#### Scenario: The functions that read weights and samples through it

- **WHEN** `stats.Skewness` is called on a nil `*DataList`
- **THEN** it returns an error prefixed `sample:`, neither crashing nor reporting `empty data`
- **AND** `WeightedMean` and `WeightedMovingAverage` no longer read their weights through `ProcessData`: they take a `[]float64`, so weights it could not read do not compile

### Requirement: The big.Rat helpers never panic

`SqrtRat` SHALL return nil for a nil or negative input. `PowRat` SHALL return the reciprocal of the positive power for a negative exponent, 1 for exponent 0, and nil for a nil base or a zero base with a negative exponent. Both SHALL be marked Deprecated and name their `math/big` replacement, as SHALL `SortTimes` (replaced by `slices.SortFunc`) and `F64orRat`.

#### Scenario: A negative square root

- **WHEN** a caller passes `big.NewRat(-1, 1)` to `SqrtRat`
- **THEN** it returns nil and does not panic

#### Scenario: A negative exponent

- **WHEN** a caller asks `PowRat(big.NewRat(2, 3), -2)`
- **THEN** it returns 9/4
