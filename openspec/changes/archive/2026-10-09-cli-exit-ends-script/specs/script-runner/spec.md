## ADDED Requirements

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
