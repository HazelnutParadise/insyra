## ADDED Requirements

### Requirement: DAY, HOUR, MINUTE and SECOND warn that they change in 0.4

`DAY`, `HOUR`, `MINUTE` and `SECOND` SHALL log one warning per process per function, naming 0.4 and `DUR(x, unit)`. Their results SHALL NOT change.

#### Scenario: Many rows
- **WHEN** `DAY(A)` is evaluated on many rows
- **THEN** one warning is logged, and every result is what it was

### Requirement: Names removed in 0.4 warn once

`TONUM`, `TOSTR` and `SUBSTR` SHALL log one warning per process per name, naming 0.4 and `VALUE`, `TEXT` and `MID`. Their results SHALL NOT change. `DAYOFMONTH` SHALL log no warning, because it has no replacement on 0.3.x.

#### Scenario: Many rows
- **WHEN** `TONUM(A)` is evaluated on many rows
- **THEN** one warning is logged, and every result is what it was

### Requirement: MOD and % warn where their answer changes in 0.4

`MOD` and `%` SHALL log one warning per process the first time a non-zero remainder's sign differs from the second number's, naming 0.4 and both answers. Their results SHALL NOT change, and other inputs SHALL NOT warn.

#### Scenario: Signs that differ
- **WHEN** `MOD(7, 3)` and then `MOD(-7, 3)` are evaluated
- **THEN** only the second warns, saying -1 becomes 2, and it still gives -1

### Requirement: WEEKDAY, TRIM and COUNT warn about what changes in 0.4

`WEEKDAY` SHALL warn once per process the first time it runs. `TRIM` SHALL warn once the first time a run of spaces stays inside its result. `COUNT` SHALL warn once the first time it counts text, a bool or `NaN`. Their results SHALL NOT change. `COUNTA` SHALL count every value that is not nil.

#### Scenario: COUNT over text
- **WHEN** `COUNT(A)` runs over a column holding numbers and text
- **THEN** one warning names 0.4 and `COUNTA`, and the count is what it was

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
