## ADDED Requirements

### Requirement: Windows are bounded before anything is allocated
`AbnormalReturns` SHALL return an error, without panicking, when an end of `EventWindow` or `EstimationWindow` is more than 2^29 rows from the event, when the table has more than 2^29 rows, or when `EventWindow` spans more rows than the table has.

#### Scenario: An extreme window
- **WHEN** `EventWindow` is `[minInt, 0]`
- **THEN** an error is returned

### Requirement: A non-finite abnormal return or CAR is missing data
An event whose abnormal return, running CAR or market-model fit is not a finite number SHALL be dropped as a whole and counted in `Missing`, and a cross-sectional mean or spread whose sum overflows SHALL give NaN statistics.

#### Scenario: A CAR that overflows
- **WHEN** an event has abnormal returns of 1e308, -1 and 1e308
- **THEN** the event is counted in `Missing` and no mean CAR is infinite
