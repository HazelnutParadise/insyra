# enum-conventions Specification

## Purpose
Every integer enum prints a name, and an error reports an enum value by name rather than by number, so a reader of a log or an error can tell which setting was meant.

## Requirements

### Requirement: Every numeric enum prints its name

Every exported enum type whose underlying type is an integer SHALL have a `String()` method. A known value SHALL print a name. An unknown value SHALL print `TypeName(n)`.

#### Scenario: Known value
- **WHEN** `quant.VaRParametric` is formatted with `%v`
- **THEN** the text is `VaRParametric`

#### Scenario: Unknown value
- **WHEN** `ml.ClassAverage(9)` is formatted with `%v`
- **THEN** the text is `ClassAverage(9)`, not `macro`

### Requirement: Errors name an enum value

An error that reports an enum value SHALL format it with `%v`.

#### Scenario: Unknown method
- **WHEN** `quant.ValueAtRisk` is called with `VaRMethod(3)`
- **THEN** the error text contains `VaRMethod(3)`
