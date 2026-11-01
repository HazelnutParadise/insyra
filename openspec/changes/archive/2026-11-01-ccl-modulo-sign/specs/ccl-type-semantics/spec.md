## ADDED Requirements

### Requirement: A remainder has the sign of the divisor

`MOD(a, b)` and `a % b` SHALL give the remainder with the sign of `b`, as Excel's `MOD` does, for integers and floats alike. They SHALL give the same answer as each other.

#### Scenario: Signs that differ
- **WHEN** `MOD(-7, 3)`, `-7 % 3`, `MOD(7, -3)` and `MOD(-7.5, 2)` are evaluated
- **THEN** they give 2, 2, -2 and 0.5
