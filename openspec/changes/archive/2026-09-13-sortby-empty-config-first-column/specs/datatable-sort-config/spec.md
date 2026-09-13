## REMOVED Requirements

### Requirement: A sort config must name a column

**Reason**: The owner ruled on 2026-09-13 that a config naming no column sorts by the first column. `SortBy` only runs when a caller asked for a sort, so the first column is not a sort nobody asked for, and refusing the empty config also refused `{ColumnNumber: 0}`, which is the same Go value.

**Migration**: None. `{}`, `{Descending: true}` and `{ColumnNumber: 0}` sort by the first column as they did before `sortby-column-selection`, and `ColumnIndex: "A"` keeps working.

## ADDED Requirements

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
