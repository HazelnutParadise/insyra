## ADDED Requirements

### Requirement: DAY, HOUR, MINUTE and SECOND warn that they change in 0.4

`DAY`, `HOUR`, `MINUTE` and `SECOND` SHALL log one warning per process per function, naming 0.4 and `DUR(x, unit)`. Their results SHALL NOT change.

#### Scenario: Many rows
- **WHEN** `DAY(A)` is evaluated on many rows
- **THEN** one warning is logged, and every result is what it was

### Requirement: DUR expresses a duration in a unit

`DUR(x, unit)` SHALL return `x` in `unit` as a number. `x` SHALL be a `time.Duration` or a string `time.ParseDuration` reads, with `d` and `w` also read as 24 and 168 hours. `unit` SHALL be required and be one of `day`, `hour`, `minute` or `second`, singular or plural, in any case.

#### Scenario: A duration string
- **WHEN** `DUR('36h', 'day')`, `DUR('1h30m', 'minute')` and `DUR('2d', 'hour')` are evaluated
- **THEN** they give 1.5, 90 and 48

### Requirement: DUR leaves an empty cell empty and refuses what it cannot read

`DUR` SHALL return nil for a nil `x`. A number, which has no unit, and text it cannot read SHALL be an error naming the value.

#### Scenario: A number and unreadable text
- **WHEN** `DUR(36, 'hour')` and `DUR('soon', 'hour')` are evaluated
- **THEN** each is an error naming the value
