## ADDED Requirements

### Requirement: WriteONNX is a Deprecated synonym of ExportONNX

`WriteONNX(w io.Writer, fitted any) error` SHALL remain for one release as a Deprecated function that exports and refuses exactly what `ExportONNX` does, and its doc comment SHALL name `ExportONNX`.

#### Scenario: The deprecated synonym writes the same bytes
- **WHEN** 對同一個已 fit 的線性模型分別呼叫 `ExportONNX` 與 `WriteONNX`
- **THEN** 兩者寫出的位元組相同，`WriteONNX` 的 doc comment 有指名 `ExportONNX` 的 `Deprecated:` 段落

#### Scenario: The deprecated synonym refuses a value that is not a model
- **WHEN** 以字串或 `nil` 呼叫 `WriteONNX`
- **THEN** 回傳錯誤，沒有寫入

### Requirement: ExportONNX keeps its any parameter

`ExportONNX(w io.Writer, fitted any) error` SHALL keep its `any` parameter on this line, so a value that is not a model is refused at run time.

#### Scenario: A value that is not a model
- **WHEN** 以字串呼叫 `ExportONNX`
- **THEN** 回傳錯誤，沒有寫入
