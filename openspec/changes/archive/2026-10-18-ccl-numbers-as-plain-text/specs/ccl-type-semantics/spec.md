## ADDED Requirements

### Requirement: CCL writes a float as plain decimal text
When `&`, `CONCAT` or a string function converts a finite `float64` or `float32` below 1e21 in magnitude to text, the system SHALL write the shortest decimal that reads back as the same number, without an exponent.

#### Scenario: A small number
- **WHEN** `'x' & A` is evaluated for A equal to 0.0000001
- **THEN** the result is `x0.0000001`

#### Scenario: Length of a literal
- **WHEN** `LEN(1000000)` is evaluated
- **THEN** the result is 7

### Requirement: Large and non-finite floats keep their spelling
A float of 1e21 or more in magnitude, NaN and the infinities SHALL keep Go's default text.

#### Scenario: 1e21
- **WHEN** a string function converts 1e21 to text
- **THEN** the text is `1e+21`
