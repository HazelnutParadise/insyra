## ADDED Requirements

### Requirement: A command answers to its aliases everywhere
`Dispatch` 與 `LookupCommand` SHALL 以指令的 `Name` 或任一 `Aliases` 找到同一個指令，在 one-shot、REPL、腳本、Go session 與 `help <alias>` 都一樣。`Register` SHALL 拒絕名稱是其他指令別名的 handler，以及任一別名已經是某個指令名稱或別名的 handler，並回傳錯誤且不註冊，讓同一個字永遠只對應一個指令。

#### Scenario: help names a command by its alias
- **WHEN** 使用者執行 `help quit`
- **THEN** 輸出 `exit` 的說明，而不是 unknown command

#### Scenario: An alias already taken
- **WHEN** 程式註冊一個指令，它的別名是另一個已註冊指令的別名或名稱
- **THEN** `Register` 回傳錯誤，該指令沒有被註冊
