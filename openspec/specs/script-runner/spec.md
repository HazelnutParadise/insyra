# script-runner Specification

## Purpose
TBD - created by archiving change cli-repl. Update Purpose after archive.
## Requirements
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

### Requirement: exit and quit end a script

`exit` 與其別名 `quit` SHALL 在 `.isr` 腳本中結束該腳本：該行之後的行 SHALL NOT 執行。`run` SHALL 以 `script ended by exit at line N` 取代 `script complete`，且 SHALL 成功回傳。由另一個腳本的 `run` 行啟動的腳本遇到 `exit` 時，這條 `run` 鏈上的每個腳本 SHALL 一起停止，控制權回到最外層 `run` 的呼叫者（one-shot 的 shell、REPL 的提示字元或 Go API 的呼叫者）。

Go `Session.ExecuteFile` SHALL 在 `exit` 或 `quit` 行停止並回傳 nil。REPL 中的 `exit` 與 `quit` SHALL 結束 REPL。在 REPL 與腳本之外執行時（one-shot `insyra exit`、單獨的 `Session.Execute("exit")`），`exit` SHALL 回傳錯誤，說明它只能結束 REPL 或腳本。`exit` 回傳的每個錯誤 SHALL 包住 `engine/dsl` 匯出的 `ErrExit`。`Dispatch` SHALL 以指令的任一別名找到該指令。

#### Scenario: Lines after exit do not run
- **WHEN** 腳本內容為 `newdl 1 as a`、`exit`、`newdl 2 as b`，以 `run` 執行
- **THEN** `a` 存在、`b` 不存在，輸出含 `script ended by exit at line 2` 而不含 `script complete`，`run` 回傳 nil

#### Scenario: quit is exit
- **WHEN** 同一份腳本以 `quit` 取代 `exit`
- **THEN** 結果相同

#### Scenario: exit in a nested script stops the chain
- **WHEN** 外層腳本 `run` 一個含 `exit` 的內層腳本，`run` 行之後還有其他行
- **THEN** 內層 `exit` 之後的行與外層 `run` 行之後的行都不執行，最外層 `run` 回傳 nil

#### Scenario: exit outside the REPL or a script
- **WHEN** 不在 REPL 也不在腳本中執行 `exit`
- **THEN** 回傳錯誤，`errors.Is(err, dsl.ErrExit)` 為真，錯誤訊息說明 `exit` 只能結束 REPL 或腳本

#### Scenario: ExecuteFile stops at exit
- **WHEN** `Session.ExecuteFile` 讀到 `exit` 行
- **THEN** 之後的行不執行，回傳 nil

