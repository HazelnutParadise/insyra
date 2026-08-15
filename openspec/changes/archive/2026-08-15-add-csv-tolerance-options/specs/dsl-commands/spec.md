# dsl-commands Delta Spec

## ADDED Requirements

### Requirement: Load CSV tolerance options

`load` 命令 SHALL 支援 CSV 專用選項 `ragged true|false` 與 `trimspace true|false`（皆預設 `false`）。`ragged true` 時系統 SHALL 以 ragged-rows 容忍模式載入（短列補空字串、長列自動加欄保留）；`trimspace true` 時 SHALL 忽略欄位前導空白（含引號前空白）。非 CSV 格式帶任一選項時 SHALL 回傳錯誤。布林值解析 SHALL 沿用既有規則（`true|false|yes|no|on|off|1|0`）。

#### Scenario: Load noisy CSV with tolerance options

- **WHEN** 使用者執行 `load "inventory.csv" ragged true trimspace true as t`，檔案含表尾註記列與引號前空白
- **THEN** 載入成功

#### Scenario: Defaults stay strict

- **WHEN** 使用者執行 `load "inventory.csv" as t`（未帶選項），檔案含欄數不齊的列
- **THEN** 回傳解析錯誤，與現行行為相同

#### Scenario: Options rejected for non-CSV

- **WHEN** 使用者執行 `load "data.json" ragged true` 或 `load "data.xlsx" sheet S trimspace true`
- **THEN** 回傳錯誤說明該選項僅適用於 CSV
