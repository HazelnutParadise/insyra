## ADDED Requirements

### Requirement: Error positions count from zero
The system SHALL name the position of an unreadable cell in a `stats` error as a zero-based index worded `row index N` or `condition index N`, matching the position `Get(N)` takes, while a group, cell or subject named by argument order keeps counting from one.

#### Scenario: The third observation is bad
- **WHEN** the third value passed to a hypothesis test is NaN
- **THEN** the error says `row index 2`, not `row 3`

#### Scenario: A group is named by argument order
- **WHEN** the second list passed to `OneWayANOVA` holds a NaN
- **THEN** the error names `group 2` and the zero-based row index of the cell
