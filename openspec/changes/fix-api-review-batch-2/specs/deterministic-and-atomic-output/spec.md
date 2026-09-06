## ADDED Requirements

### Requirement: lp additional-info table has a fixed row order

`SolveFromFile`／`SolveModel` 的第二張表 SHALL 依 `Status, Execution Time, Warnings, Full Output, Iterations, Nodes` 順序排列。

#### Scenario: Row order
- **WHEN** 任一次求解（含錯誤路徑）
- **THEN** 列名依上述順序

### Requirement: File geocode cache is written atomically

`fileGeocodeCache.Set` SHALL 先寫入暫存檔再 rename 到目標路徑。

#### Scenario: Cache file after Set
- **WHEN** `Set` 之後讀取目錄
- **THEN** 只有目標檔，無殘留 `.tmp`，內容可解析
