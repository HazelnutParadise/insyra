## ADDED Requirements

### Requirement: Date part functions read a date on 0.3.x

`DAY`, `HOUR`, `MINUTE` and `SECOND` SHALL give that part of a date or of text that reads as one. A duration, text that reads as a duration, and a number SHALL convert as before and SHALL warn once that the form goes away in 0.4. `DATEPART` and `DAYOFMONTH` SHALL warn once, naming what to use.

#### Scenario: A date

- **WHEN** an expression is `HOUR('2024-01-02T06:30:15Z')`
- **THEN** the result is 6

#### Scenario: A duration

- **WHEN** an expression is `HOUR('36h')`
- **THEN** the result is 36, and a warning names `DUR`
