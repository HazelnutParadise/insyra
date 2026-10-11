## ADDED Requirements

### Requirement: DATEPART gives one part of a date

`DATEPART(d, unit)` SHALL return, as a `float64`, the year, month, day of the month, hour, minute or second of the date `d`, read in `d`'s own time zone and dropping any fraction of a second. The unit is a string, singular or plural, in any letter case. An unknown unit, a `d` that is not a date, or a number of arguments other than two SHALL be an error.

#### Scenario: Each part
- **WHEN** 欄 `A` 存 `'2024-03-09T06:07:08Z'`，求值 `DATEPART(A, 'year')`、`'month'`、`'day'`、`'hour'`、`'minute'`、`'second'`
- **THEN** 分別得到 2024、3、9、6、7、8

#### Scenario: The date's own time zone
- **WHEN** 欄 `B` 存 UTC+8 的 2024-12-31 23:59:58.999，求值 `DATEPART(B, 'hour')` 與 `DATEPART(B, 'second')`
- **THEN** 分別得到 23 與 58

#### Scenario: An unknown unit
- **WHEN** 求值 `DATEPART(A, 'week')`
- **THEN** 回傳錯誤
