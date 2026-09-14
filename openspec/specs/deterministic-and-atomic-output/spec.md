# deterministic-and-atomic-output Specification

## Purpose
`lp`'s additional-info table has a fixed row order and the file geocode cache is written atomically (temp file + rename).

## Requirements
### Requirement: lp additional-info table has a fixed row order

`SolveFromFile`／`SolveModel` 回傳第二張表時，該表 SHALL 依 `Status, Execution Time, Warnings, Full Output, Iterations, Nodes` 順序排列。

#### Scenario: Row order
- **WHEN** 任一次回傳第二張表的求解，包含以這張表回報錯誤的路徑（例如 GLPK 失敗或寫入 LP 檔失敗）
- **THEN** 列名依上述順序
- **AND** 參數錯誤或建立暫存檔失敗時回傳 `nil, nil`，沒有這張表

### Requirement: File geocode cache is written atomically

`fileGeocodeCache.Set` SHALL 先寫入暫存檔再 rename 到目標路徑。

#### Scenario: Cache file after Set
- **WHEN** `Set` 之後讀取目錄
- **THEN** 只有目標檔，無殘留 `.tmp`，內容可解析

