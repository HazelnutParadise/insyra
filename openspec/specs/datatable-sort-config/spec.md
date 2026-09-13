# datatable-sort-config Specification

## Purpose
定義 `DataTable.SortBy` 如何從 `DataTableSortConfig` 選出要排序的欄：沒有指定欄位的設定依第一欄排序，同時指定多種方式時依索引、名稱、數字的順序選欄並警告，指向不存在的欄時整次排序不動並把錯誤記在 `SortBy`。
## Requirements
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

### Requirement: A config that names no column sorts by the first column

`SortBy` 收到沒有設定 `ColumnIndex`、`ColumnName`，且 `ColumnNumber` 為零的設定時 SHALL 依第一欄排序，SHALL NOT 在 `Err()` 記錄錯誤，也 SHALL NOT 記錄警告。因為 `{ColumnNumber: 0}` 與空設定在 Go 中是同一個值，兩者 SHALL 行為相同。設定了 `ColumnIndex` 或 `ColumnName` 時 SHALL 依該欄排序，SHALL NOT 因 `ColumnNumber` 的零值改排第一欄。

#### Scenario: An empty config
- **WHEN** 以 `DataTableSortConfig{}` 或 `{ColumnNumber: 0}` 呼叫 `SortBy`
- **THEN** 依第一欄遞增排序，`Err()` 為 nil，沒有警告

#### Scenario: Only Descending
- **WHEN** 以只設 `Descending: true` 的設定呼叫 `SortBy`
- **THEN** 依第一欄遞減排序，`Err()` 為 nil，沒有警告

#### Scenario: A name with ColumnNumber left at zero
- **WHEN** 只設定 `ColumnName`
- **THEN** 依該名稱的欄排序，沒有警告

