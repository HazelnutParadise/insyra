## ADDED Requirements

### Requirement: Group keys compare integers by value

`GroupBy`, `Pivot`, `Merge` keys and `OpNUnique` SHALL put integers of equal value in one group whatever their Go width: `int`, `int8` through `int64`, `uint` through `uint64` and `uintptr`. `float32` and `float64` values SHALL group by value among themselves. An integer SHALL NOT share a group with a float or a string of the same value.

#### Scenario: Eleven widths and two floats

- **WHEN** a column holds the value 1 in each of the eleven integer widths, as `float64` and as `float32`, and as the string `"1"`, and is grouped
- **THEN** there are three groups, of sizes 11, 2 and 1

#### Scenario: A distinct count

- **WHEN** `OpNUnique` runs over `1`, `int64(1)`, `uintptr(1)`, `1.0` and `"1"`
- **THEN** it returns 3
