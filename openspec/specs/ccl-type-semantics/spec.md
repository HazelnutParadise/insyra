# ccl-type-semantics Specification

## Purpose
CCL never produces a value for an operation it cannot perform. Values that have no common ordering are an error rather than a silent false, an index that is not a whole number is an error rather than a truncation, a cell of every Go numeric type is a number, and the evaluator's internal representations never reach a cell.
## Requirements
### Requirement: Strings have an ordering

When neither operand of `<`, `>`, `<=` or `>=` converts to a number and both are strings, CCL SHALL compare them lexicographically. When one operand converts to a number and the other is a string that does not, CCL SHALL return an error. `nil` compared for size against any value SHALL remain false, as documented.

#### Scenario: Two words
- **WHEN** 求值 `'abc' < 'abd'`
- **THEN** 得到 true，且 `'abc' > 'abd'` 得到 false

#### Scenario: A word against a number
- **WHEN** 求值 `'hello' > 5`
- **THEN** 回傳錯誤而不是 false

### Requirement: nil concatenates as the empty string

`&` and `CONCAT` SHALL render `nil` as the empty string. No Go formatting placeholder SHALL appear in a cell.

#### Scenario: Concatenating nil
- **WHEN** 求值 `nil & 'x'`
- **THEN** 得到 `"x"`

### Requirement: Internal types never reach a cell

No internal representation SHALL become a cell value. A column range used as a whole result SHALL resolve to the current row restricted to those columns, the same way a bare column reference resolves to the current row's value. A row range SHALL remain an error, because it names rows but not what they are rows of, and the message SHALL say how to attach it to a column. `LAG` and `LEAD` SHALL accept a row-shaped argument (`@` or a column range); every other sequence function SHALL refuse one, because it does arithmetic on each element.

#### Scenario: A bare column range
- **WHEN** 執行 `AddColUsingCCL("r", "A:B")`
- **THEN** 每一格是該列的 A、B 值（`[]any`），與 `(A:B).#` 相同，SHALL NOT 出現 `ColumnRange`

#### Scenario: A sequence function over the whole row
- **WHEN** 求值 `LAG(@, 1)`
- **THEN** 每一列得到前一列的內容，與 `LAG(@.#, 1)` 相同，第一列為 nil

#### Scenario: A sequence function that needs numbers
- **WHEN** 求值 `CUMSUM(@)`
- **THEN** 回傳錯誤說明整列不是數字

#### Scenario: A bare row range
- **WHEN** 求值 `1:2`
- **THEN** 回傳錯誤，訊息指出要接在欄位上（`A.(1:2)`）

#### Scenario: The consumers of a range are unchanged
- **WHEN** 求值 `SUM(A:C)`、`SUM((A:C).(2:5))` 與 `SUM(A.(0:1))`
- **THEN** 結果與批次 8 之前相同，範圍內的每個儲存格只計入一次

### Requirement: AND and OR check their arguments

`AND()` and `OR()` SHALL require at least two arguments, and SHALL return an error for an argument that cannot be converted to a boolean, matching `&&` and `||`.

#### Scenario: No arguments
- **WHEN** 求值 `AND()`
- **THEN** 回傳錯誤而不是 true

#### Scenario: A non-boolean argument
- **WHEN** 求值 `AND('abc', true)`
- **THEN** 回傳錯誤，與 `'abc' && true` 一致

### Requirement: Indices and windows are whole numbers

A row index, a range bound and a sequence-function window or period SHALL be a finite integer. A fractional, NaN or infinite value SHALL be an error, SHALL NOT be truncated, and SHALL NOT depend on the CPU architecture.

#### Scenario: A fractional row index
- **WHEN** 求值 `A.(1.7)`
- **THEN** 回傳錯誤而不是第 1 列的值

#### Scenario: A fractional rolling window
- **WHEN** 求值 `ROLLING_MEAN(A, 2.9)`
- **THEN** 回傳錯誤而不是以視窗 2 計算

### Requirement: Fractional days keep their fraction

Adding or subtracting a number of days to a date SHALL preserve sub-hour precision.

#### Scenario: A thousandth of a day
- **WHEN** 對日期欄求值 `A + 0.001`
- **THEN** 時間戳前進 86.4 秒

### Requirement: A literal that cannot be represented is an error

數字字面值 SHALL 在超出 float64 範圍時回報編譯錯誤，SHALL NOT 靜默變成 `+Inf`。指數形式（`1e5`、`1.5e-3`、`2E+3`）SHALL 是合法的數字字面值；`e` 後面沒有數字時 SHALL 仍視為識別字，讓名為 `E` 或 `E1` 的欄位不受影響。

#### Scenario: A literal too large for a float64
- **WHEN** 運算式是 400 個 9
- **THEN** 回報超出範圍的編譯錯誤，而不是每列都得到 `+Inf`

#### Scenario: Exponent notation
- **WHEN** 運算式是 `1e5 + 1`
- **THEN** 得到 100001

### Requirement: A format that does not fit is an error

`TOSTR`／`TEXT` 的兩引數形式 SHALL 在格式與值不相符時回報錯誤，SHALL NOT 把 `fmt` 的錯誤標記（`%!d(...)`、`%!(NOVERB)`）當成結果寫進儲存格。原本就出現在值的文字或格式字串裡的相同文字 SHALL NOT 被當成錯誤標記。

#### Scenario: A verb that does not fit
- **WHEN** `TOSTR(1.5, '%d')`
- **THEN** 回報錯誤，指出格式與值的型別

#### Scenario: Text that only looks like a marker
- **WHEN** 對值 `Item (MISSING)` 求值 `TOSTR(A, '%s')`，或求值 `TOSTR(3, 'n=%v (MISSING)')`
- **THEN** 分別得到 `Item (MISSING)` 與 `n=3 (MISSING)`，不回報錯誤

### Requirement: Every Go numeric type is a number

CCL SHALL read a cell of every Go integer type (`int`, `int8`, `int16`, `int32`, `int64`, `uint`, `uint8`, `uint16`, `uint32`, `uint64`) and of `float32` as a number, in arithmetic, comparisons, conditions, function arguments, aggregates and row indices, giving the result it gives for the same value held as an `int64`. A `uint64` above the `int64` range SHALL be read as the nearest `float64`. A `float32` NaN SHALL be missing to `ISNA` and `IFNA`.

#### Scenario: Aggregates over narrow integers
- **WHEN** 欄 `A` 依序存 `int16(3)`、`int16(4)`、`uint8(5)`，求值 `SUM(A)`、`MAX(A)`、`AVG(A)`
- **THEN** 分別得到 12、5、4，不回傳錯誤

#### Scenario: Comparing a narrow integer
- **WHEN** 欄 `A` 存 `uint8(1)`，逐列求值 `A == 1`
- **THEN** 得到 true

#### Scenario: Arithmetic and conditions on a narrow integer
- **WHEN** 欄 `A` 存 `int8(3)`，逐列求值 `A * 2` 與 `IF(A, 'y', 'n')`
- **THEN** 分別得到 6 與 `'y'`

#### Scenario: A row index held as an integer column
- **WHEN** 欄 `B` 存 `int64(1)`，逐列求值 `A.B`
- **THEN** 得到 `A` 第 1 列（從 0 數）的值；`B` 存其他整數型別或 `float32(1)` 時結果相同，存 `float32(1.5)` 時回傳錯誤，與 `float64` 的列號規則相同

#### Scenario: A uint64 past the float64 range of exact integers
- **WHEN** 欄 `A` 存 `uint64(1<<53 + 1)`，求值 `A + 0`
- **THEN** 得到 `int64(1<<53 + 1)`，與 `int64(1<<53 + 1)` 的結果相同

#### Scenario: A float32 NaN is missing
- **WHEN** 欄 `A` 存 `float32(NaN)`，求值 `ISNA(A)` 與 `IFNA(A, 0)`
- **THEN** 分別得到 true 與 0

### Requirement: A doubled quote inside a literal is one quote

A string literal SHALL be enclosed in single or double quotes. Inside it, the enclosing quote character written twice SHALL stand for one such character, and the other quote character SHALL stand for itself. A bracketed column name SHALL follow the same rule. A backslash SHALL be an ordinary character. Splitting a statement-mode script into statements SHALL treat a doubled quote as part of the literal, so a `;` or a line break inside the literal does not end the statement.

#### Scenario: Single-quoted literal holding a single quote
- **WHEN** 求值 `'it''s'`
- **THEN** 得到字串 `it's`

#### Scenario: Double-quoted literal holding double quotes
- **WHEN** 求值 `"say ""hi"""`
- **THEN** 得到字串 `say "hi"`

#### Scenario: A literal that is one quote
- **WHEN** 求值 `''''`
- **THEN** 得到字串 `'`

#### Scenario: Bracketed column name with a quote
- **WHEN** 表格有名為 `O'Brien` 的欄，求值 `['O''Brien']`
- **THEN** 得到該欄的值

#### Scenario: A backslash does not escape
- **WHEN** 編譯 `'it\'s'`
- **THEN** 回傳 `unclosed string` 錯誤

#### Scenario: Statement splitting keeps the literal whole
- **WHEN** 以 statement mode 編譯 `NEW('x') = 'a;b''c` 換行 `d'` 換行 `NEW('it''s') = 1`
- **THEN** 得到兩個敘述，第一個的字串是 `a;b'c` 換行 `d`，第二個建立名為 `it's` 的欄

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

### Requirement: Integers stay integers

A number literal written as digits alone, with an optional leading minus, that fits an `int64` SHALL be an `int64`; any other number literal SHALL be a `float64`. When both operands are integers — a cell of any Go integer type whose value fits an `int64`, or an integer literal — `+`, `-`, `*`, `%` and unary minus SHALL compute in `int64` and give an `int64`, and `==`, `!=`, `<`, `>`, `<=` and `>=` SHALL compare them exactly. In arithmetic a `nil` next to an integer SHALL be an integer `0`. An integer result outside the `int64` range SHALL be an error that says it overflows. `SUM`, `MIN` and `MAX`, and their streaming forms, SHALL give an `int64` when every value they use is an integer, and an integer `SUM` outside the `int64` range SHALL be an error. `MOD` of two integers SHALL give an `int64`. The row index `#` SHALL be an `int64`. Every other numeric operation SHALL compute in `float64` as before, `/` and `^` included.

#### Scenario: An ID past 2^53 keeps its digits
- **WHEN** 欄 `A` 存 `int64(9007199254740993)`，求值 `A + 0`、`A * 1`、`A - 1`
- **THEN** 分別得到 `int64(9007199254740993)`、`int64(9007199254740993)`、`int64(9007199254740992)`

#### Scenario: Integer literals
- **WHEN** 求值 `1 + 2`、`7 % 3`、`-5`、`1.0`、`1e3`
- **THEN** 分別得到 `int64(3)`、`int64(1)`、`int64(-5)`、`float64(1)`、`float64(1000)`

#### Scenario: Division and powers stay float64
- **WHEN** 求值 `7 / 2`、`6 / 3`、`2 ^ 3`
- **THEN** 分別得到 `float64` 的 3.5、2、8

#### Scenario: Exact comparison
- **WHEN** 欄 `A` 存 `int64(1<<53 + 1)`、欄 `B` 存 `int64(1<<53)`，求值 `A == B` 與 `A > B`
- **THEN** 分別得到 false 與 true

#### Scenario: Overflow is an error
- **WHEN** 欄 `A` 存 `int64(math.MaxInt64)`，求值 `A + 1`、`A * 2`、`SUM(A, 1)`
- **THEN** 都回傳提到 overflow 的錯誤

#### Scenario: Aggregates of integers
- **WHEN** 欄 `A` 依序存 `int64(1<<53 + 1)`、`int64(1<<53)`、nil、`int64(2)`，求值 `SUM(A)`、`MAX(A)`、`MIN(A)`
- **THEN** 分別得到 `int64(1<<54 + 3)`、`int64(1<<53 + 1)`、`int64(2)`；streaming 形式得到相同的值與型別

#### Scenario: A decimal makes it float64
- **WHEN** 欄 `A` 依序存 `int64(1)`、`2.5`、`int64(3)`，求值 `SUM(A)`
- **THEN** 得到 `float64(6.5)`

#### Scenario: nil next to an integer
- **WHEN** 欄 `C` 存 `int16(3)`，求值 `nil + C` 與 `nil + 2.5`
- **THEN** 分別得到 `int64(3)` 與 `float64(2.5)`

### Requirement: A computed column holds one kind of number

A column written by `AddColUsingCCL`, `EditColByIndexUsingCCL`, `EditColByNameUsingCCL` or a statement of `ExecuteCCL` SHALL hold one kind of number: when the numbers among its values include a `float64` or `float32`, every value of a Go integer type whose magnitude is at most 2^53 SHALL become the `float64` of the same value. An integer of larger magnitude SHALL keep its type and value. A column whose numbers are all integers SHALL be left as computed. Values that are not numbers SHALL be left as computed.

#### Scenario: A literal fallback in a float column
- **WHEN** 欄 `A` 為 `["19.5", "abc", "30"]`，`AddColUsingCCL("r", "COALESCE(TONUM(A), 0)")`
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

### Requirement: DATEPART gives one part of a date

`DATEPART(d, unit)` SHALL return, as a `float64`, the year, month, day of the month, hour, minute or second of `d` for a unit of `year`, `month`, `day`, `hour`, `minute` or `second`, in either singular or plural form and in any letter case, read in `d`'s own time zone, dropping any fraction of a second. `d` SHALL be a `time.Time` or a string Insyra's date parser reads as a date. Any other unit, a `d` that is not a date, a unit that is not a string, or a number of arguments other than two SHALL be an error.

#### Scenario: Each part
- **WHEN** 欄 `A` 存 `'2024-03-09T06:07:08Z'`，求值 `DATEPART(A, 'year')`、`'month'`、`'day'`、`'hour'`、`'minute'`、`'second'`
- **THEN** 分別得到 2024、3、9、6、7、8

#### Scenario: The date's own time zone
- **WHEN** 欄 `B` 存 UTC+8 的 2024-12-31 23:59:58.999，求值 `DATEPART(B, 'hour')` 與 `DATEPART(B, 'second')`
- **THEN** 分別得到 23 與 58

#### Scenario: An unknown unit
- **WHEN** 求值 `DATEPART(A, 'week')`
- **THEN** 回傳錯誤

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

