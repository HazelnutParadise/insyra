# instance-error-contract Specification

## Purpose
實例錯誤契約：`Err()` 黏住第一個錯誤、`PopErr()` 讀後即清，只有真正的失敗才被記錄。

## Requirements
### Requirement: Err() is sticky

`DataList`／`DataTable`／`isr` 物件上第一個被記錄的錯誤 SHALL 保留到 `ClearErr()` 或 `PopErr()` 被呼叫為止；後續錯誤 SHALL NOT 覆寫它。`Clone()` 產生的物件 SHALL 從無錯誤開始。

#### Scenario: First error survives a chain
- **WHEN** 一個串接中第一步與第三步都失敗
- **THEN** `Err()` 回報第一步的錯誤

### Requirement: PopErr reads and clears

`PopErr()` SHALL 回傳目前的錯誤並清除它；再次呼叫 SHALL 回傳 nil。`IDataList` 與 `IDataTable` SHALL 包含 `PopErr()`。

#### Scenario: Pop then reuse
- **WHEN** 對有錯誤的物件呼叫 `PopErr()` 兩次
- **THEN** 第一次非 nil，第二次為 nil

### Requirement: A lookup that finds nothing is not an error

讀取類操作找不到結果或輸入為空時 SHALL 以 Warning 記錄並 SHALL NOT 設定 `Err()`；包含 `Get` 越界、`FindFirst`／`FindLast` 找不到、`FindAll`／`Count` 於空 list、空 list 的統計量、`GetCol`／`GetColByName`／`GetRow`／`GetRowByName` 找不到。

#### Scenario: Missing value keeps Err() clean
- **WHEN** 對沒有該值的 list 呼叫 `FindFirst`
- **THEN** 回傳 nil 且 `Err()` 為 nil

