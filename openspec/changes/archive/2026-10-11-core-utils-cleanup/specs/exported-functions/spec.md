## ADDED Requirements

### Requirement: The big.Rat helpers never panic

`SqrtRat` SHALL return nil for a nil or negative input. `PowRat` SHALL return the reciprocal of the positive power for a negative exponent, 1 for exponent 0, and nil for a nil base or a zero base with a negative exponent. Both SHALL be marked Deprecated and name their `math/big` replacement, as SHALL `SortTimes` (replaced by `slices.SortStableFunc`) and `F64orRat`, and all SHALL be noted as removed in 0.4.

#### Scenario: A negative square root

- **WHEN** a caller passes `big.NewRat(-1, 1)` to `SqrtRat`
- **THEN** it returns nil and does not panic

#### Scenario: A negative exponent

- **WHEN** a caller asks `PowRat(big.NewRat(2, 3), -2)`
- **THEN** it returns 9/4
