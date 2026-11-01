## ADDED Requirements

### Requirement: ROUND reads a number as it is written

`ROUND` SHALL round half away from zero on the shortest decimal text of its number, not on the binary value, and SHALL keep `NaN` and infinities as they are. A result of zero SHALL NOT be negative.

#### Scenario: A number just below its written value in binary

- **WHEN** an expression is `ROUND(1.005, 2)`
- **THEN** the result is 1.01

#### Scenario: Negative digits

- **WHEN** an expression is `ROUND(1250, -2)`
- **THEN** the result is 1300
