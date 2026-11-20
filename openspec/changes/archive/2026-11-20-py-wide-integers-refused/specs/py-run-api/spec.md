## MODIFIED Requirements

### Requirement: An integer bound into an integer field is exact

Bound into an integer field, an integer SHALL be exact whenever the field's type holds it. An integer that neither an `int64` nor a `uint64` holds SHALL be an error for an integer field.

#### Scenario: A large id into an int64
- **WHEN** Python returns `2**53 + 1` and the result type is `int64`
- **THEN** the result is 9007199254740993

#### Scenario: Just below the int64 range
- **WHEN** Python returns `-(2**63) - 1` and the result type is `int64`
- **THEN** the call returns an error saying the integer fits neither an `int64` nor a `uint64`

## ADDED Requirements

### Requirement: An integer beyond 64 bits is never rounded silently

An integer that neither an `int64` nor a `uint64` holds SHALL come back as the nearest value of a `float32` or `float64` it is bound into, because the result type asked for a float. Anywhere else, an `any`, a table or list cell, a map or slice of `any`, or an integer field, the call SHALL return an error that names the integer, shortened to 30 characters, and says to send `float(x)` or `str(x)`. An integer too large for a float64 SHALL be an error wherever it is bound.

#### Scenario: Into an any
- **WHEN** Python returns `2**64 + 4` and the result type is `any`
- **THEN** the call returns an error saying the integer fits neither an `int64` nor a `uint64`

#### Scenario: Into a float64
- **WHEN** Python returns `2**64 + 4` and the result type is `float64`
- **THEN** the result is 1.8446744073709552e+19 and there is no error

#### Scenario: In a table cell
- **WHEN** Python returns a DataFrame with a cell holding `2**64 + 4` and the result type is `*insyra.DataTable`
- **THEN** the call returns the same error
