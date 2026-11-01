## ADDED Requirements

### Requirement: DUR expresses a duration in a unit

`DUR(x, unit)` SHALL return `x` in `unit` as a number. `x` SHALL be a `time.Duration` or a string `time.ParseDuration` reads, with `d` and `w` also read as 24 and 168 hours. `unit` SHALL be required and be one of `day`, `hour`, `minute` or `second`, singular or plural, in any case.

#### Scenario: A duration string
- **WHEN** `DUR('36h', 'day')`, `DUR('1h30m', 'minute')` and `DUR('2d', 'hour')` are evaluated
- **THEN** they give 1.5, 90 and 48

#### Scenario: The difference of two dates
- **WHEN** `A` is 2024-01-03 12:00 and `B` is 2024-01-02 00:00, and `DUR(A - B, 'hour')` is evaluated
- **THEN** it gives 36, as `DATEDIFF(A, B, 'hour')` does

### Requirement: DUR leaves an empty cell empty and refuses what it cannot read

`DUR` SHALL return nil for a nil `x`. A number, which has no unit, and text it cannot read SHALL be an error naming the value.

#### Scenario: An empty cell
- **WHEN** `A` is nil and `DUR(A, 'day')` is evaluated
- **THEN** it gives nil with no error

#### Scenario: A number and unreadable text
- **WHEN** `DUR(36, 'hour')` and `DUR('soon', 'hour')` are evaluated
- **THEN** each is an error naming the value

## MODIFIED Requirements

### Requirement: DAY, HOUR, MINUTE and SECOND refuse a duration

A `time.Duration`, a string `time.ParseDuration` accepts, or a number given to `DAY`, `HOUR`, `MINUTE` or `SECOND` SHALL be an error that says the function takes a date and names `DUR(x, unit)` and `DATEDIFF(end, start, unit)`. A nil value SHALL be an error saying the value is empty. Any other value that is not a date, or a number of arguments other than one, SHALL be an error.

#### Scenario: A duration is refused
- **WHEN** 欄 `A`、`B` 為日期，求值 `DAY(A - B)`、`HOUR(7200)`、`DAY('36h')`
- **THEN** 都回傳錯誤，訊息說明需要日期，並提到 `DUR(x, '<unit>')` 與 `DATEDIFF(end, start, '<unit>')`

#### Scenario: What the error points to
- **WHEN** A 為 2024-01-03 12:00、B 為 2024-01-02 00:00，求值 `DATEDIFF(A, B, 'day')` 與 `(A - B) / 3600`
- **THEN** 分別得到 1.5 與 36

#### Scenario: An empty cell
- **WHEN** `A` is nil and `DAY(A)` is evaluated
- **THEN** the error says the value is empty, not that it is a number
