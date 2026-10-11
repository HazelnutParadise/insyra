## ADDED Requirements

### Requirement: An empty list converts to an empty Series

`pd.FromDataList` SHALL return an empty `*Series` with a nil error for a list with no elements.

#### Scenario: Empty list
- **WHEN** `pd.FromDataList(insyra.NewDataList())` 被呼叫
- **THEN** 回傳非 nil 的 `*Series`，其 `Len()` 為 0，`DType()` 為 `interface {}`，且沒有錯誤

#### Scenario: Empty Series back to a list
- **WHEN** 由 `FromDataList` 得到的空 Series 以 `ToDataList` 轉回
- **THEN** 回傳的 `*insyra.DataList` 其 `Len()` 為 0，且沒有錯誤

### Requirement: A nil list is still an error

`pd.FromDataList` SHALL return an error for a nil list.

#### Scenario: Nil list
- **WHEN** `pd.FromDataList(nil)` 被呼叫
- **THEN** 回傳 nil 的 `*Series` 與一個錯誤
