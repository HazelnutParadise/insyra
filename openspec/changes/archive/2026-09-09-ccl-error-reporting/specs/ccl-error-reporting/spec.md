## ADDED Requirements

### Requirement: A failure says which phase it happened in

A CCL failure SHALL identify itself as a compile failure or an evaluation failure. The two SHALL NOT share a message prefix that hides the difference.

#### Scenario: A malformed expression
- **WHEN** 執行 `AddColUsingCCL("r", "SUM(A B)")`
- **THEN** 錯誤訊息說明這是編譯失敗，而不是與執行期失敗共用同一個前綴

#### Scenario: A failure on one row
- **WHEN** 某一列使 `A / B` 除以零
- **THEN** 錯誤訊息說明這是執行期失敗

### Requirement: An evaluation failure names its row

An evaluation failure SHALL report the row index it happened on. An expression that does not depend on the row SHALL say so instead of reporting a misleading row.

#### Scenario: A division by zero on the middle row
- **WHEN** 欄 B 的第 1 列是 0，求值 `A / B`
- **THEN** 錯誤訊息指出第 1 列

### Requirement: A position is always a byte offset

Every compile error that reports a position SHALL report a byte offset into the expression the caller wrote, and SHALL include the source text at that offset. A token index SHALL NOT be reported as a position.

#### Scenario: A missing comma
- **WHEN** 編譯 `SUM(A B)`
- **THEN** 錯誤指出的偏移量落在運算式字串中 `B` 的位置，並附上該處的文字

### Requirement: No Go internals appear in a message

A CCL error message SHALL NOT contain a Go struct dump or a pointer value.

#### Scenario: An unexpected token
- **WHEN** 編譯 `1 +)`
- **THEN** 錯誤訊息包含該 token 的文字，SHALL NOT 包含 `{` 或 `0x`

### Requirement: Errors carry a type callers can match

Compile and evaluation failures SHALL be values of exported types carrying the expression, the position or row, and (for evaluation) the wrapped cause. `errors.As` SHALL reach them through a `DataTable`'s recorded error.

#### Scenario: Reacting to a compile failure
- **WHEN** `AddColUsingCCL` 因為運算式無法編譯而失敗
- **THEN** `errors.As(dt.Err(), &compileErr)` 為真，且 `compileErr.Expr` 是原始運算式
