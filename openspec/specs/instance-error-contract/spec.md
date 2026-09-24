# instance-error-contract Specification

## Purpose
`DataList`／`DataTable` 實例錯誤的讀取與記錄契約：`Err()` 保存最近一次錯誤，`PopErr()` 讀取並清除，`SetErr()` 讓包裝套件以與核心相同的方式記錄錯誤。

## Requirements
### Requirement: PopErr reads and clears

`PopErr()` SHALL 回傳目前的錯誤並清除它；再次呼叫 SHALL 回傳 nil。`IDataList` 與 `IDataTable` SHALL 包含 `PopErr()`。

#### Scenario: Pop then reuse
- **WHEN** 對有錯誤的物件呼叫 `PopErr()` 兩次
- **THEN** 第一次非 nil，第二次為 nil

### Requirement: SetErr records like an internal warning

`SetErr(packageName, funcName, msg, args...)` SHALL 以 Warning 等級寫一筆日誌與全域緩衝區紀錄，SHALL 以該錯誤取代實例目前的 `Err()`，並 SHALL 回傳接收者。`IDataList` 與 `IDataTable` SHALL 包含 `SetErr()`。

#### Scenario: Later SetErr wins
- **WHEN** 對同一實例先後呼叫兩次 `SetErr`
- **THEN** `Err()` 回報第二次的錯誤，等級為 `LogLevelWarning`
