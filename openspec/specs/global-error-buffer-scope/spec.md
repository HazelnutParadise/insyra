# global-error-buffer-scope Specification

## Purpose
全域錯誤緩衝區的定位：有上限的診斷日誌，不是錯誤處理 API。

## Requirements
### Requirement: The global buffer is a bounded diagnostic log

全域錯誤緩衝區 SHALL 有上限（1536 筆），超過時 SHALL 丟棄最舊的紀錄而非無限成長。它 SHALL 被文件定位為診斷工具；錯誤處理 SHALL 走實例 `Err()` 或函式回傳的 `error`。

#### Scenario: Bounded growth
- **WHEN** 記錄 5000 筆錯誤且無人讀取
- **THEN** `GetErrorCount()` 不超過 1536

### Requirement: Instance and global records are independent

從全域緩衝區 pop 一筆錯誤 SHALL NOT 影響任何實例的 `Err()`；`ClearErr()` SHALL NOT 影響全域緩衝區。

#### Scenario: Pop does not clear instance error
- **WHEN** 一次失敗同時記入實例與全域，然後呼叫 `PopAllErrors()`
- **THEN** 實例的 `Err()` 仍非 nil

