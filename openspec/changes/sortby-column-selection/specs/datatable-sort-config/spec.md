## ADDED Requirements

### Requirement: A sort config must name a column

`SortBy` 收到沒有指定任何欄位的設定時 SHALL 在 `Err()` 記錄以 `SortBy` 為名的錯誤，且 SHALL NOT 改變表格。因為 `{ColumnNumber: 0}` 與空設定在 Go 中是同一個值，單獨的 `{ColumnNumber: 0}` SHALL 同樣被拒絕，錯誤訊息 SHALL 指出第一欄應以 `ColumnIndex: "A"` 指定。

#### Scenario: An empty config
- **WHEN** 以 `DataTableSortConfig{}` 或只設 `Descending` 的設定呼叫 `SortBy`
- **THEN** `Err()` 記錄錯誤，表格不變

#### Scenario: The first column by position
- **WHEN** 以 `ColumnIndex: "A"` 排序
- **THEN** 依第一欄排序，沒有錯誤

### Requirement: A column that is not there is refused

設定指向的欄位不存在時，`SortBy` SHALL 記錄以 `SortBy` 為名的錯誤，SHALL NOT 讓錯誤指向內部查找函式，且 SHALL NOT 改變表格。

#### Scenario: An index, a name or a number that matches nothing
- **WHEN** `ColumnIndex` 找不到欄、`ColumnName` 不存在，或 `ColumnNumber` 超出範圍
- **THEN** `Err()` 記錄的錯誤指名 `SortBy`，表格不變

### Requirement: A multi-level sort is all or nothing

多層排序中只要有一層的設定無效，`SortBy` SHALL NOT 套用其餘任何一層。

#### Scenario: One bad level among good ones
- **WHEN** 第一層有效、第二層指向不存在的欄
- **THEN** 表格完全不變，`Err()` 記錄第二層的錯誤

### Requirement: More than one selector follows precedence with a warning

同一個設定同時指定多個欄位時，`SortBy` SHALL 依「`ColumnIndex`、`ColumnName`、`ColumnNumber`」的順序選欄並完成排序，SHALL 記錄警告指出被忽略的欄位，且 SHALL NOT 在 `Err()` 記錄錯誤。`ColumnNumber` 只有在不為零時才視為有指定。

#### Scenario: An index and a name together
- **WHEN** 同時設定 `ColumnIndex` 與 `ColumnName`
- **THEN** 依 `ColumnIndex` 排序，記錄警告，`Err()` 為 nil

#### Scenario: A name and a number together
- **WHEN** 同時設定 `ColumnName` 與非零的 `ColumnNumber`
- **THEN** 依 `ColumnName` 排序，記錄警告，`Err()` 為 nil
