## REMOVED Requirements

### Requirement: Duration functions refuse a date

**Reason**: `DAY`, `HOUR`, `MINUTE` and `SECOND` now take a part of a date, as in Excel, instead of converting a duration.

**Migration**: use `DATEDIFF(end, start, unit)`, or divide the difference of two dates (which counts seconds), where these functions were given a duration: `DAY(A - B)` becomes `DATEDIFF(A, B, 'day')`.

## ADDED Requirements

### Requirement: DAY, HOUR, MINUTE and SECOND take a part of a date

`DAY`, `HOUR`, `MINUTE` and `SECOND` SHALL return, as a `float64`, the day of the month, hour, minute or second of a `time.Time` or of a string Insyra's date parser reads as a date, read in the date's own time zone and dropping any fraction of a second, giving what `DATEPART(d, 'day')`, `'hour'`, `'minute'` and `'second'` give. A `time.Duration`, a string `time.ParseDuration` accepts, or a number SHALL be an error that says the function takes a date and names `DATEDIFF(end, start, unit)`. Any other value that is not a date, or a number of arguments other than one, SHALL be an error. `DAYOFMONTH` SHALL keep giving what `DAY` gives.

#### Scenario: Parts of a date
- **WHEN** 欄 `A` 存 UTC 的 2024-01-02 06:30:15.999，求值 `DAY(A)`、`HOUR(A)`、`MINUTE(A)`、`SECOND(A)`
- **THEN** 分別得到 2、6、30、15

#### Scenario: A date string in another time zone
- **WHEN** 求值 `DAY('2024-12-31T23:59:58+08:00')` 與 `HOUR('2024-12-31T23:59:58+08:00')`
- **THEN** 分別得到 31 與 23

#### Scenario: A duration is refused
- **WHEN** 欄 `A`、`B` 為日期，求值 `DAY(A - B)`、`HOUR(7200)`、`DAY('36h')`
- **THEN** 都回傳錯誤，訊息說明需要日期，並提到 `DATEDIFF(end, start, '<unit>')`

#### Scenario: What the error points to
- **WHEN** A 為 2024-01-03 12:00、B 為 2024-01-02 00:00，求值 `DATEDIFF(A, B, 'day')` 與 `(A - B) / 3600`
- **THEN** 分別得到 1.5 與 36
