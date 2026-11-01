## ADDED Requirements

### Requirement: A removed function name says what to write

`TONUM`, `TOSTR`, `SUBSTR` and `DAYOFMONTH`, with any arguments, SHALL be an error that says the name was removed and names `VALUE`, `TEXT`, `MID` and `DAY` respectively.

#### Scenario: A formula still using one
- **WHEN** `TONUM(A)` is evaluated
- **THEN** the error says it was removed and to use `VALUE`

## MODIFIED Requirements

### Requirement: A format that does not fit is an error

`TEXT` 的兩引數形式 SHALL 在格式與值不相符時回報錯誤，SHALL NOT 把 `fmt` 的錯誤標記（`%!d(...)`、`%!(NOVERB)`）當成結果寫進儲存格。原本就出現在值的文字或格式字串裡的相同文字 SHALL NOT 被當成錯誤標記。

#### Scenario: A verb that does not fit
- **WHEN** `TEXT(1.5, '%d')`
- **THEN** 回報錯誤，指出格式與值的型別

#### Scenario: Text that only looks like a marker
- **WHEN** 對值 `Item (MISSING)` 求值 `TEXT(A, '%s')`，或求值 `TEXT(3, 'n=%v (MISSING)')`
- **THEN** 分別得到 `Item (MISSING)` 與 `n=3 (MISSING)`，不回報錯誤

### Requirement: A computed column holds one kind of number

A column written by `AddColUsingCCL`, `EditColByIndexUsingCCL`, `EditColByNameUsingCCL` or a statement of `ExecuteCCL` SHALL hold one kind of number: when the numbers among its values include a `float64` or `float32`, every value of a Go integer type whose magnitude is at most 2^53 SHALL become the `float64` of the same value. An integer of larger magnitude SHALL keep its type and value. A column whose numbers are all integers, and every value that is not a number, SHALL be left as computed.

#### Scenario: A literal fallback in a float column
- **WHEN** 欄 `A` 為 `["19.5", "abc", "30"]`，`AddColUsingCCL("r", "COALESCE(VALUE(A), 0)")`
- **THEN** `r` 是 `float64` 的 `[19.5, 0, 30]`

#### Scenario: Integers stay integers
- **WHEN** 欄 `A` 為 `[int64(1), nil, int64(3)]`，`AddColUsingCCL("r", "A + 1")`
- **THEN** `r` 是 `int64` 的 `[2, 1, 4]`

#### Scenario: Only numbers are settled
- **WHEN** 欄 `A` 為 `int64` 的 `[0, 1, 2]`，`ExecuteCCL("NEW('r') = IF(A == 0, 'none', IF(A == 1, 2.5, 3))")`
- **THEN** `r` 是 `["none", 2.5, float64(3)]`

#### Scenario: An integer a float64 cannot hold
- **WHEN** 欄 `A` 為 `[int64(1<<53 + 1), int64(2), int64(3)]`，`AddColUsingCCL("r", "IF(# == 2, 0.5, A)")`
- **THEN** `r` 是 `[int64(1<<53 + 1), float64(2), 0.5]`

#### Scenario: A copied integer column keeps its types
- **WHEN** 欄 `A` 為 `int16` 的 `[1, 2]`，`AddColUsingCCL("b", "A")`
- **THEN** `b` 仍是 `int16` 的 `[1, 2]`

### Requirement: DAY, HOUR, MINUTE and SECOND take a part of a date

`DAY`, `HOUR`, `MINUTE` and `SECOND` SHALL return, as a `float64`, the day of the month, hour, minute or second of a `time.Time` or of a string Insyra's date parser reads as a date, read in the date's own time zone and dropping any fraction of a second, giving what `DATEPART(d, 'day')`, `'hour'`, `'minute'` and `'second'` give. `DAY` SHALL keep giving what `DAY` gives.

#### Scenario: Parts of a date
- **WHEN** 欄 `A` 存 UTC 的 2024-01-02 06:30:15.999，求值 `DAY(A)`、`HOUR(A)`、`MINUTE(A)`、`SECOND(A)`
- **THEN** 分別得到 2、6、30、15

#### Scenario: A date string in another time zone
- **WHEN** 求值 `DAY('2024-12-31T23:59:58+08:00')` 與 `HOUR('2024-12-31T23:59:58+08:00')`
- **THEN** 分別得到 31 與 23
