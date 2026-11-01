## REMOVED Requirements

### Requirement: DATEPART gives one part of a date

**Reason**: `YEAR`, `MONTH`, `DAY`, `HOUR`, `MINUTE` and `SECOND` read dates, so `DATEPART` only repeated them under a longer name.

**Migration**: Write `HOUR(d)` for `DATEPART(d, 'hour')`, and likewise for the other units.

## ADDED Requirements

### Requirement: A formula using DATEPART is told what to write

`DATEPART`, with any arguments, SHALL be an error that says it was removed and names `YEAR`, `MONTH`, `DAY`, `HOUR`, `MINUTE` and `SECOND`.

#### Scenario: A formula still using it
- **WHEN** `DATEPART(A, 'hour')` is evaluated
- **THEN** the error says it was removed and names `HOUR(d)`
