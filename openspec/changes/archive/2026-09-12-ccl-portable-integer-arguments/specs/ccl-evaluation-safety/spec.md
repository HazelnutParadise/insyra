## ADDED Requirements

### Requirement: Numeric arguments give the same answer on every platform

When CCL turns a numeric argument into an integer or a duration, the result SHALL NOT depend on the platform. NaN, infinities, a count or shift outside the int64 range, and a value that overflows a `time.Duration` SHALL be refused. The exception is a character count, character position or digit count: it SHALL be clamped, so that a count past the end of a string still means "to the end".

#### Scenario: A huge length
- **WHEN** 在 amd64 或 arm64 上求值 `MID('abc', 2, 10^300)`
- **THEN** 兩者都得到 `"bc"`

#### Scenario: A huge date shift
- **WHEN** 求值 `DATEADD(D, 10^300, 'day')` 或 `D + 10^300`
- **THEN** 回傳錯誤，不產生日期

#### Scenario: A large but deterministic argument
- **WHEN** 求值 `LAG(A, 3000000000)`、`LEN(REPEAT('', 100000000))` 或 `DATEADD(D, 3000000000, 'day')`
- **THEN** 分別得到整欄 nil、`0` 與一個日期，不回傳錯誤

#### Scenario: A negative fractional repeat count
- **WHEN** 求值 `REPEAT('ab', 0-0.5)` 與 `REPEAT('ab', 0-1)`
- **THEN** 前者捨去小數成 0 次，得到 `""`，與原本相同；後者回傳錯誤
