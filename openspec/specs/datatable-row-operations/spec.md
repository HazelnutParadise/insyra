# datatable-row-operations Specification

## Purpose
Pins the documented semantics of `DropRowsByIndex` (negative and duplicate indices), `Transpose` (all row names carried over), `ChangeRowName` (no name stealing) and `AppendRowsByColIndex` (the table grows to the addressed column).
## Requirements
### Requirement: DropRowsByIndex normalises and de-duplicates

`DropRowsByIndex` SHALL 先把負索引換算成正索引、去除重複與越界者，再由大到小刪除。

#### Scenario: Negative and positive indices together
- **WHEN** `[0,1,2,3]` 的表呼叫 `DropRowsByIndex(-1, 0)`
- **THEN** 剩 `[1, 2]`

#### Scenario: Duplicate index
- **WHEN** `[0,1,2,3]` 的表呼叫 `DropRowsByIndex(1, 1)`
- **THEN** 剩 `[0, 2, 3]`

### Requirement: Transpose keeps every row name

`Transpose` SHALL 把第 i 個列名變成第 i 個欄名，i 涵蓋所有列而不受原欄數限制。

#### Scenario: More rows than columns
- **WHEN** 2 欄 3 列、列名 r0/r1/r2 的表 `Transpose()`
- **THEN** 欄名為 `[r0, r1, r2]`

### Requirement: ChangeRowName cannot steal another row's name

`ChangeRowName(old, new)` 在 `new` 已存在時 SHALL 經 `safeRowName` 取得不衝突的名稱，其他列的名稱 SHALL NOT 改變。

#### Scenario: Rename onto an existing name
- **WHEN** 列名 `[a, b]` 的表呼叫 `ChangeRowName("b", "a")`
- **THEN** 第 0 列仍是 `a`，第 1 列為 `a_1`

### Requirement: AppendRowsByColIndex never drops a value

`AppendRowsByColIndex` 遇到超出現有寬度的索引時 SHALL 補足到該索引為止的欄位，值 SHALL 寫入該欄。

#### Scenario: Index beyond current width
- **WHEN** 兩欄表呼叫 `AppendRowsByColIndex(map[string]any{"Z": 42})`
- **THEN** 表有 26 欄，`GetElement(-1, "Z")` 為 42

### Requirement: Unnamed columns merge by position

垂直合併 SHALL 以名稱對齊有名稱的欄位，以「在無名欄中的位置」對齊沒有名稱的欄位。空字串 SHALL NOT 被視為重複的名稱。

#### Scenario: Two tables built without column names
- **WHEN** 兩張以 `NewDataTable(NewDataList(...), NewDataList(...))` 建立的表垂直合併
- **THEN** 合併成功，兩欄各自依位置接起來，而不是回報「重複欄名 ""」

