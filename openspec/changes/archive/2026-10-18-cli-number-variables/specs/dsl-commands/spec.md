## ADDED Requirements

### Requirement: Numeric arguments may name a number variable
The system SHALL read a numeric command argument that does not parse as a number as the name of a variable holding a number, and SHALL use that value, reporting the usual invalid-argument error when no such variable exists.

#### Scenario: A variable supplies the number
- **WHEN** `n` holds 2 and `shift x n` runs
- **THEN** the shift uses 2 periods

#### Scenario: A literal number is unchanged
- **WHEN** an argument parses as a number
- **THEN** it is used as that number even if a variable has the same text as its name

### Requirement: Statistic commands can store their result
The system SHALL store the result of `sum`, `mean`, `median`, `mode`, `stdev`, `var`, `min`, `max` and `range` under the variable named by a trailing `as <var>`, and SHALL print the number of `show <var>` for a number variable.

#### Scenario: Mean stored and shown
- **WHEN** `mean x as m` then `show m` run
- **THEN** `m` holds the mean and `show m` prints it
