# global-error-buffer-scope Specification

## Purpose
全域錯誤緩衝區的定位與上限：它是容量固定、保留最近紀錄的診斷日誌，不是錯誤處理 API；錯誤處理走實例 `Err()`／`PopErr()` 或函式回傳的 `error`。

## Requirements
### Requirement: The global buffer is a bounded diagnostic log

全域錯誤緩衝區 SHALL 以 `ErrorBufferCapacity`（1536 筆）為上限，超過時 SHALL 丟棄最舊的紀錄而非無限成長。它 SHALL 被文件定位為診斷工具；`PopError`、`PopErrorByPackageName`、`PopErrorByFuncName`、`PopErrorAndCallback`、`PeekError`、`GetErrorsByLevel`、`GetErrorsByPackage`、`PopErrorInfo`、`HasErrorAboveLevel` SHALL 標為 Deprecated，並指向 `GetAllErrors`、`PopAllErrors`、`HasError`、`GetErrorCount`、`ClearErrors`。

#### Scenario: Bounded growth
- **WHEN** 記錄 5000 筆錯誤且無人讀取
- **THEN** `GetErrorCount()` 不超過 1536，且最新的一筆仍在緩衝區中

### Requirement: Instance and global records are independent

從全域緩衝區 pop 一筆錯誤 SHALL NOT 影響任何實例的 `Err()`；`ClearErr()` SHALL NOT 影響全域緩衝區。

#### Scenario: Pop does not clear instance error
- **WHEN** 一次失敗同時記入實例與全域，然後呼叫 `PopAllErrors()`
- **THEN** 實例的 `Err()` 仍非 nil
