## ADDED Requirements

### Requirement: DATEADD by months stops at the month's end

`DATEADD(d, n, 'month')` and `DATEADD(d, n, 'year')` SHALL move `d` by `n` months or `n` years and keep its day of the month, and SHALL give the target month's last day when that month has fewer days, as Excel's `EDATE` and pandas' `DateOffset` do. The time of day and the time zone of `d` SHALL be kept.

#### Scenario: One month after January 31 in a leap year
- **WHEN** 求值 `DATEADD('2024-01-31', 1, 'month')`
- **THEN** 得到 2024-02-29

#### Scenario: Backwards into a short month
- **WHEN** 求值 `DATEADD('2024-03-31', -1, 'month')`
- **THEN** 得到 2024-02-29

#### Scenario: One year after February 29
- **WHEN** 求值 `DATEADD('2024-02-29', 1, 'year')`
- **THEN** 得到 2025-02-28

#### Scenario: The time of day is kept
- **WHEN** 求值 `DATEADD('2024-05-31 10:30:15', 1, 'month')`
- **THEN** 得到 2024-06-30 10:30:15

#### Scenario: Days are not months
- **WHEN** 求值 `DATEADD('2024-01-31', 31, 'day')`
- **THEN** 得到 2024-03-02
