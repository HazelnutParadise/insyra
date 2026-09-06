## ADDED Requirements

### Requirement: A failed command never stores nil

`col`／`row`／`movavg`／`expsmooth`／`diff` 找不到或算不出結果時 SHALL 回錯誤且 SHALL NOT 寫入變數；`SaveState` 遇到 typed nil SHALL 不 panic。

#### Scenario: Unknown column
- **WHEN** 執行 `col dt nope as c`
- **THEN** 回錯誤且 `c` 不存在

### Requirement: NaN survives persistence

含 NaN／±Inf 的 DataList 與 DataTable SHALL 能 `SaveState` 並 `RestoreVariables` 回相同型別與值（NaN 對 NaN）；舊格式（字串 JSON）的 DataTable SHALL 仍可讀。

#### Scenario: Table with a blank cell
- **WHEN** 儲存含 `[1, NaN, 3]` 欄的 DataTable 後還原
- **THEN** 變數仍是 `*DataTable`，該欄第二格是 NaN

### Requirement: Root flags and scripts

`--env`／`--no-color`／`--log-level` 放在 `newdl`／`addcol`／`addrow`／`show` 前面 SHALL 生效。`run` 期間 `env open` SHALL 只切換環境；`run` 巢狀超過 16 層 SHALL 回錯誤。

#### Scenario: Flags before newdl
- **WHEN** 執行 `insyra --env e2 newdl 1 2 3 as ex`
- **THEN** `ex` 有 3 個元素且只存在於 e2
