## MODIFIED Requirements

### Requirement: ProcessData reports failure as an error

`ProcessData` SHALL return `([]any, error)`. It SHALL read a slice or an array element by element, an `IDataList` by its values, and a pointer to a slice or an array by what it points to. It SHALL return an error and a nil slice for any other type, for a nil input and for a nil pointer, a nil `*DataList` included, and SHALL NOT panic. An empty slice SHALL give an empty, non-nil slice and a nil error.

#### Scenario: A value it cannot read

- **WHEN** a caller passes `42`, `nil` or a nil `*DataList`
- **THEN** `ProcessData` returns a nil slice and a non-nil error

#### Scenario: The functions that read weights and samples through it

- **WHEN** `stats.Skewness` is called on a nil `*DataList`
- **THEN** it returns an error prefixed `sample:`, neither crashing nor reporting `empty data`
- **AND** `WeightedMean` and `WeightedMovingAverage` no longer read their weights through `ProcessData`: they take a `[]float64`, so weights it could not read do not compile
