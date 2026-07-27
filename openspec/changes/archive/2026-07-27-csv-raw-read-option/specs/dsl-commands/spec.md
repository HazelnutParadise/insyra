# dsl-commands Delta Spec

## ADDED Requirements

### Requirement: Load CSV without type inference
`load` 命令 SHALL 支援 CSV 專用選項 `infer true|false`(預設 `true`)。`infer false` 時系統 SHALL 以 raw-strings 模式載入,所有 cell 保留原始字串。非 CSV 格式帶 `infer` 選項時 SHALL 回傳錯誤。布林值解析 SHALL 沿用既有規則(`true|false|yes|no|on|off|1|0`)。

#### Scenario: Load CSV with inference disabled
- **WHEN** 使用者執行 `load "stocks.csv" infer false as t`
- **THEN** 系統以 raw-strings 模式載入,股票代號 `0050` 保留為字串

#### Scenario: Default keeps inference on
- **WHEN** 使用者執行 `load "data.csv" as t`(未帶 `infer`)
- **THEN** 系統照常執行欄位型別推斷

#### Scenario: infer rejected for non-CSV
- **WHEN** 使用者執行 `load "data.json" infer false` 或 `load "data.xlsx" sheet S infer false`
- **THEN** 系統回傳選項不適用的錯誤
