# ccl-type-semantics Specification

## Purpose
CCL 遇到無法表示的數字字面值或與值不相符的格式時回報錯誤，不把 `+Inf` 或 `fmt` 的錯誤標記當成結果寫進儲存格。
## Requirements
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

### Requirement: Every Go numeric type is a number

CCL SHALL read a cell of every Go integer type and of `float32` as a number, in arithmetic, comparisons, conditions, function arguments, aggregates and row indices, giving the result it gives for the same value held as an `int64`. A row index of any numeric type SHALL drop its fractional part and SHALL be an error when it is `NaN`, infinite or outside the `int32` range. A `float32` NaN SHALL be missing to `ISNA` and `IFNA`.

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
- **THEN** 得到 `A` 第 1 列（從 0 數）的值；`B` 存其他整數型別或 `float32(1)` 時結果相同，存 `float32(1.5)` 時小數部分捨去，也得到第 1 列，與 `float64` 的列號規則相同

#### Scenario: A row index outside the countable range
- **WHEN** 欄 `B` 存 `float64(NaN)`、`float64(+Inf)` 或 `float64(1<<31)`，逐列求值 `A.B`
- **THEN** 回傳錯誤

#### Scenario: A uint64 past the float64 range of exact integers
- **WHEN** 欄 `A` 存 `uint64(1<<53 + 1)`，求值 `A + 0`
- **THEN** 得到 `float64(1<<53)`，與 `int64(1<<53 + 1)` 的結果相同

#### Scenario: A float32 NaN is missing
- **WHEN** 欄 `A` 存 `float32(NaN)`，求值 `ISNA(A)` 與 `IFNA(A, 0)`
- **THEN** 分別得到 true 與 0

### Requirement: Internal types never reach a cell

No internal representation SHALL become a cell value. A column range used as a whole result SHALL resolve to the current row restricted to those columns, the same way a bare column reference resolves to the current row's value. A row range SHALL remain an error, because it names rows but not what they are rows of, and the message SHALL say how to attach it to a column.

#### Scenario: A bare column range
- **WHEN** 執行 `AddColUsingCCL("r", "A:B")`
- **THEN** 每一格是該列的 A、B 值（`[]any`），與 `(A:B).#` 相同，SHALL NOT 出現 `ColumnRange`

#### Scenario: A bare row range
- **WHEN** 求值 `1:2`
- **THEN** 回傳錯誤，訊息指出要接在欄位上（`A.(1:2)`）

#### Scenario: The consumers of a range are unchanged
- **WHEN** 求值 `SUM(A:C)`、`SUM((A:C).(2:5))` 與 `SUM(A.(0:1))`
- **THEN** 結果與本變更之前相同，範圍內的每個儲存格只計入一次

### Requirement: LAG and LEAD accept a whole row

`LAG` and `LEAD` SHALL accept a row-shaped argument (`@` or a column range), giving each row the content of the row before or after it. Every other sequence function SHALL refuse one, because it does arithmetic on each element.

#### Scenario: A sequence function over the whole row
- **WHEN** 求值 `LAG(@, 1)`
- **THEN** 每一列得到前一列的內容，與 `LAG(@.#, 1)` 相同，第一列為 nil

#### Scenario: A sequence function that needs numbers
- **WHEN** 求值 `CUMSUM(@)`
- **THEN** 回傳錯誤說明整列不是數字

### Requirement: CCL writes a float as plain decimal text
When `&`, `CONCAT` or a string function converts a finite `float64` or `float32` below 1e21 in magnitude to text, the system SHALL write the shortest decimal that reads back as the same number, without an exponent.

#### Scenario: A small number
- **WHEN** `'x' & A` is evaluated for A equal to 0.0000001
- **THEN** the result is `x0.0000001`

#### Scenario: Length of a literal
- **WHEN** `LEN(1000000)` is evaluated
- **THEN** the result is 7

### Requirement: Large and non-finite floats keep their spelling
A float of 1e21 or more in magnitude, NaN and the infinities SHALL keep Go's default text.

#### Scenario: 1e21
- **WHEN** a string function converts 1e21 to text
- **THEN** the text is `1e+21`

