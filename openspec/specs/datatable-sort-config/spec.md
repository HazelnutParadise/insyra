# datatable-sort-config Specification

## Purpose
定義 `DataTable.SortBy` 如何從 `DataTableSortConfig` 選出要排序的欄：沒有指定欄位的設定依第一欄排序，同時指定多種方式時依索引、名稱、數字的順序選欄並警告，指向不存在的欄時整次排序不動並把錯誤記在 `SortBy`。
## Requirements
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

