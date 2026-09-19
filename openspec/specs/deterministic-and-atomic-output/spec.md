# deterministic-and-atomic-output Specification

## Purpose
The additional-info table that `lp`'s `SolveFromFile` and `SolveModel` return lists its rows in a fixed order, so the table reads the same on every run.

## Requirements
### Requirement: lp additional-info table has a fixed row order

`SolveFromFile`／`SolveModel` 回傳第二張表時，該表 SHALL 依 `Status, Execution Time, Warnings, Full Output, Iterations, Nodes` 順序排列。

#### Scenario: Row order
- **WHEN** 任一次回傳第二張表的求解，包含以這張表回報錯誤的路徑（例如 GLPK 失敗或寫入 LP 檔失敗）
- **THEN** 列名依上述順序
- **AND** 參數錯誤或建立暫存檔失敗時回傳 `nil, nil`，沒有這張表

