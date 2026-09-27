## MODIFIED Requirements

### Requirement: NaN survives persistence

含 NaN／±Inf 的 DataList、DataTable 與頂層純量 SHALL 能 `SaveState` 並 `RestoreVariables` 回相同型別與值（NaN 對 NaN），含這些值的環境 SHALL 能 `env export`。先前版本寫入的 state.json（DataTable 為 `ToJSON_String(true)` 字串，或含 `$float` 標記的欄位陣列）SHALL 仍可讀。

#### Scenario: Table with a blank cell
- **WHEN** 儲存含 `[1, NaN, 3]` 欄的 DataTable 後還原
- **THEN** 變數仍是 `*DataTable`，該欄第二格是 NaN

#### Scenario: Table without special floats
- **WHEN** 儲存不含 NaN／±Inf 的 DataTable
- **THEN** 它與含 NaN 的表以相同的帶型別格式寫入，還原後欄位順序與每一格的型別不變

#### Scenario: Legacy table layouts
- **WHEN** 還原先前版本寫入的 state.json，其中一個 DataTable 以 `ToJSON_String(true)` 字串儲存，另一個以含 `$float` 標記的欄位陣列儲存
- **THEN** 兩者都還原為 `*DataTable`，標記為 NaN 的格子是 NaN

#### Scenario: Export an environment holding NaN
- **WHEN** 環境中有值為 NaN 的純量變數，執行 `env export`
- **THEN** 匯出成功
