## ADDED Requirements

### Requirement: A nil list is an error, not a panic
A `stats` function that reads an `IDataList` SHALL return an error, and SHALL NOT panic in any goroutine, when that list is nil, whether bare or inside a wrapper such as `isr.DL`.

#### Scenario: A nil response in a regression
- **WHEN** `LinearRegression` is given a nil `*DataList` as y
- **THEN** it returns an error and does not panic

#### Scenario: A wrapped nil list
- **WHEN** `Skewness` is given `isr.DL`
- **THEN** it returns an error and does not panic
