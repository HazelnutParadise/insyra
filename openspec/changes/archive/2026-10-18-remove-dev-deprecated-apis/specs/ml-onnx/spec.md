## REMOVED Requirements

### Requirement: One export function, typed by the protocol
**Reason**: It named deprecated APIs that were removed in 0.4, and its scenarios cannot be edited in place; it is restated under a new name in this change.
**Migration**: See the requirement "ExportONNX is the one export function".

## ADDED Requirements

### Requirement: ExportONNX is the one export function

`ml.ExportONNX(w io.Writer, fitted Model) error` SHALL be the package-level export function. A nil `Model`, or a `Model` holding a nil pointer, SHALL be refused with an error before anything is written. `WriteONNX` SHALL NOT exist.

#### Scenario: The export function takes a model
- **WHEN** 檢查 `ml.ExportONNX` 的第二個參數型別
- **THEN** 是 `ml.Model`

#### Scenario: A nil model
- **WHEN** 以 `nil` 呼叫 `ExportONNX`
- **THEN** 回傳錯誤，沒有寫入
