## ADDED Requirements

### Requirement: One export function, typed by the protocol

`ml.ExportONNX(w io.Writer, fitted Model) error` SHALL be the package-level export function. A nil `Model`, or a `Model` holding a nil pointer, SHALL be refused with an error before anything is written. `WriteONNX(w io.Writer, fitted any) error` SHALL remain for one release as a Deprecated function whose doc comment names `ExportONNX`: a `Model` SHALL be exported exactly as `ExportONNX` exports it, and any other value SHALL be refused with an error naming its type, before anything is written.

#### Scenario: The export function takes a model
- **WHEN** 檢查 `ml.ExportONNX` 的第二個參數型別
- **THEN** 是 `ml.Model`

#### Scenario: The deprecated synonym writes the same bytes
- **WHEN** 對同一個已 fit 的線性模型分別呼叫 `ExportONNX` 與 `WriteONNX`
- **THEN** 兩者寫出的位元組相同，`WriteONNX` 的 doc comment 有指名 `ExportONNX` 的 `Deprecated:` 段落

#### Scenario: The deprecated synonym refuses a value that is not a model
- **WHEN** 以字串或 `nil` 呼叫 `WriteONNX`
- **THEN** 回傳錯誤，沒有寫入

#### Scenario: A nil model
- **WHEN** 以 `nil` 呼叫 `ExportONNX`
- **THEN** 回傳錯誤，沒有寫入
