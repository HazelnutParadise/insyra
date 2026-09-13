## ADDED Requirements

### Requirement: Numeric arguments give the same answer on every platform

When CCL turns a numeric argument into an integer or a duration, the result SHALL NOT depend on the platform. NaN, infinities and values a `time.Duration` or a date shift cannot hold SHALL be refused. The exception is a character count, character position or digit count: it SHALL be clamped, so that a count past the end of a string still means "to the end".

#### Scenario: A huge length
- **WHEN** 在 amd64 或 arm64 上求值 `MID('abc', 2, 10^300)`
- **THEN** 兩者都得到 `"bc"`

#### Scenario: A huge date shift
- **WHEN** 求值 `DATEADD(D, 10^300, 'day')` 或 `D + 10^300`
- **THEN** 回傳錯誤，不產生日期
