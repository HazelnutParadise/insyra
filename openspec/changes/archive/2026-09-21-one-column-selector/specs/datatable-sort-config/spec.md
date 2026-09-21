## MODIFIED Requirements

### Requirement: A column that is not there is refused

設定指向的欄位不存在時，`SortBy` SHALL 記錄以 `SortBy` 為名的錯誤，SHALL NOT 讓錯誤指向內部查找函式，且 SHALL NOT 改變表格。

#### Scenario: An index, a name or a number that matches nothing
- **WHEN** `Col` 是找不到的索引、不存在的 `Name`，或超出範圍的數字
- **THEN** `Err()` 記錄的錯誤指名 `SortBy`，表格不變

### Requirement: A multi-level sort is all or nothing

多層排序中只要有一層的設定無效，`SortBy` SHALL NOT 套用其餘任何一層。

#### Scenario: One bad level among good ones
- **WHEN** 第一層有效、第二層指向不存在的欄
- **THEN** 表格完全不變，`Err()` 記錄第二層的錯誤

## ADDED Requirements

### Requirement: A config that picks no column sorts by the first column

`SortBy` 收到 `Col` 為 nil 的設定時 SHALL 依第一欄排序，SHALL NOT 在 `Err()` 記錄錯誤，也 SHALL NOT 記錄警告。`DataTableSortConfig{}`、只設 `Descending` 的設定，以及 `{Col: 0}` SHALL 行為相同。

#### Scenario: An empty config
- **WHEN** 以 `DataTableSortConfig{}` 呼叫 `SortBy`
- **THEN** 依第一欄遞增排序，`Err()` 為 nil，沒有警告

#### Scenario: Only Descending
- **WHEN** 以只設 `Descending: true` 的設定呼叫 `SortBy`
- **THEN** 依第一欄遞減排序，`Err()` 為 nil，沒有警告

#### Scenario: A position of zero
- **WHEN** 以 `{Col: 0}` 呼叫 `SortBy`
- **THEN** 依第一欄排序，沒有警告，與空設定行為相同

## REMOVED Requirements

### Requirement: A config that names no column sorts by the first column

**Reason**: 這條要求的內容是 `ColumnNumber` 的零值與 `ColumnIndex`／`ColumnName` 之間怎麼互動，而那三個欄位已經收成一個 `Col`，零值不再與任何欄位競爭。同名的新要求以 `Col` 為準重寫。

**Migration**: 行為不變，空設定仍然依第一欄排序。只設定名稱的設定改寫成 `{Col: insyra.Name("score")}`。


### Requirement: More than one selector follows precedence with a warning

**Reason**: `DataTableSortConfig` 只剩一個 `Col` 欄位，同時指定多種方式在型別上已經不可能，優先順序與它的警告沒有對象。

**Migration**: `DataTableSortConfig{ColumnIndex: "B"}` 改寫成 `{Col: "B"}`，`{ColumnName: "price"}` 改寫成 `{Col: insyra.Name("price")}`，`{ColumnNumber: 2}` 改寫成 `{Col: 2}`。同時指定多個欄位的設定原本只有第一個生效，改寫時保留那一個即可。
