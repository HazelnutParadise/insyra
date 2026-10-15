## ADDED Requirements

### Requirement: Mode returns every mode in ascending order

`DataList.Mode()` SHALL return every numeric value whose count equals the highest count, in ascending order, counting cells by their `float64` value and skipping `nil` and `NaN`. When every value occurs equally often it SHALL return every value. A list with no numeric values SHALL return nil without recording an error.

#### Scenario: Every value ties
- **WHEN** `Mode()` is called on `[1, 2, 3]`, `[1, 1, 2, 2]` and `[7]`
- **THEN** it returns `[1 2 3]`, `[1 2]` and `[7]`

#### Scenario: Stable order
- **WHEN** `Mode()` is called repeatedly on `[3, 3, 1, 1, 2]`
- **THEN** it returns `[1 3]` every time

#### Scenario: Missing values are skipped
- **WHEN** `Mode()` is called on `[NaN, NaN, 1]`
- **THEN** it returns `[1]`
