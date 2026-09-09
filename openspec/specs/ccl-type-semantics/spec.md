# ccl-type-semantics Specification

## Purpose
CCL never produces a value for an operation it cannot perform. Values that have no common ordering are an error rather than a silent false, an index that is not a whole number is an error rather than a truncation, and the evaluator's internal representations never reach a cell.

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
