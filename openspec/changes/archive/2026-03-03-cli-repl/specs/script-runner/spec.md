## ADDED Requirements

### Requirement: Script file execution
系統 SHALL 提供 `run` 命令執行 `.isr` 腳本檔。

#### Scenario: Run script file
- **WHEN** 使用者執行 `run "analysis.isr"`
- **THEN** 系統讀取檔案，逐行送入 command registry 執行

#### Scenario: Script with comments
- **WHEN** 腳本檔包含以 `#` 開頭的行
- **THEN** 系統跳過註解行

#### Scenario: Script with blank lines
- **WHEN** 腳本檔包含空行
- **THEN** 系統跳過空行

#### Scenario: Script error handling
- **WHEN** 腳本中某一行執行失敗
- **THEN** 系統顯示錯誤訊息（含行號），並繼續執行後續行（或依設定停止）

#### Scenario: Script variable scope
- **WHEN** 腳本中使用 `load "data.csv" as t1` 再使用 `show t1`
- **THEN** 腳本共享同一個 ExecContext，變數在腳本行之間可用

#### Scenario: Run script from CLI
- **WHEN** 使用者執行 `insyra run "analysis.isr"`
- **THEN** 系統在當前環境下執行腳本，執行完畢後存回環境狀態
