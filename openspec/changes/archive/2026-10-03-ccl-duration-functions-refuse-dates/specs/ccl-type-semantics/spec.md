## ADDED Requirements

### Requirement: Duration functions refuse a date

`DAY`, `HOUR`, `MINUTE` and `SECOND` SHALL convert a `time.Duration`, a string `time.ParseDuration` accepts, or a number of seconds, to days, hours, minutes or seconds. A `time.Time` value or a string Insyra's date parser reads as a date SHALL be an error that says the function converts a duration and names the function that gives the corresponding part of a date: `DAYOFMONTH(x)` for `DAY`, and `TONUM(FORMAT_DATE(x, '15'))`, `TONUM(FORMAT_DATE(x, '04'))` and `TONUM(FORMAT_DATE(x, '05'))` for `HOUR`, `MINUTE` and `SECOND`.

#### Scenario: A date string reaches DAY
- **WHEN** 求值 `DAY('2024-01-02T06:00:00Z')`
- **THEN** 回傳錯誤，訊息說明 `DAY` 換算的是時間長度，並提到 `DAYOFMONTH(`

#### Scenario: A date column reaches HOUR
- **WHEN** 欄 `A` 存 `time.Time` 2024-01-02 06:30:15 UTC，逐列求值 `HOUR(A)`
- **THEN** 回傳錯誤，訊息提到 `FORMAT_DATE(` 與 `'15'`

#### Scenario: The replacement gives the Excel part
- **WHEN** 欄 `A` 存 `'2024-01-02T06:30:15Z'`，求值 `DAYOFMONTH(A)`、`TONUM(FORMAT_DATE(A, '15'))`、`TONUM(FORMAT_DATE(A, '04'))`、`TONUM(FORMAT_DATE(A, '05'))`
- **THEN** 分別得到 2、6、30、15

#### Scenario: Durations still convert
- **WHEN** 求值 `DAY(A - B)`（A 為 2024-01-03 12:00、B 為 2024-01-02 00:00）、`DAY('36h')`、`MINUTE('90s')`、`HOUR(7200)`
- **THEN** 分別得到 1.5、1.5、1.5、2，不回傳錯誤
