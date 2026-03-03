## ADDED Requirements

### Requirement: REPL interactive loop
系統 SHALL 提供互動式 REPL 迴圈，使用 readline 行編輯庫，支援行編輯、歷史瀏覽和信號處理。

#### Scenario: REPL prompt display
- **WHEN** REPL 啟動
- **THEN** 顯示 prompt 格式為 `insyra [env-name] > `（如 `insyra [default] > `）

#### Scenario: Command execution in REPL
- **WHEN** 使用者在 REPL 中輸入一條命令（如 `show t1`）並按 Enter
- **THEN** 系統執行該命令並顯示結果，然後繼續等待下一條輸入

#### Scenario: Empty line and comments
- **WHEN** 使用者輸入空行或以 `#` 開頭的行
- **THEN** 系統跳過該行，不執行任何操作

#### Scenario: Exit REPL
- **WHEN** 使用者輸入 `exit` 或 `quit` 或按 Ctrl+D
- **THEN** 系統自動儲存當前環境狀態，然後退出 REPL

#### Scenario: Ctrl+C handling
- **WHEN** 使用者在 REPL 中按 Ctrl+C
- **THEN** 系統取消當前輸入行，顯示新的 prompt（不退出 REPL）

#### Scenario: State persistence on exit
- **WHEN** REPL 退出（exit/quit/Ctrl+D）
- **THEN** 系統將當前所有變數自動序列化存入環境的 `state.json`，命令歷史存入 `history.txt`

#### Scenario: State restore on start
- **WHEN** REPL 啟動並載入一個環境
- **THEN** 系統自動從 `state.json` 反序列化還原變數，並從 `history.txt` 載入命令歷史

### Requirement: Tab completion
系統 SHALL 支援 Tab 鍵自動補全。

#### Scenario: Command name completion
- **WHEN** 使用者輸入部分命令名（如 `lo`）並按 Tab
- **THEN** 系統補全為 `load`（或顯示匹配列表）

#### Scenario: Variable name completion
- **WHEN** 使用者在需要變數名的位置按 Tab
- **THEN** 系統列出當前環境中所有變數名

#### Scenario: File path completion
- **WHEN** 使用者在 `load` 命令後按 Tab
- **THEN** 系統補全檔案路徑

### Requirement: Command history
系統 SHALL 支援上下方向鍵瀏覽命令歷史。

#### Scenario: History navigation
- **WHEN** 使用者按上方向鍵
- **THEN** 顯示上一條命令

#### Scenario: History persistence
- **WHEN** REPL 退出後重新進入同一環境
- **THEN** 之前的命令歷史仍可用

### Requirement: Clear screen
系統 SHALL 提供 `clear` 命令清除螢幕。

#### Scenario: Clear screen
- **WHEN** 使用者輸入 `clear`
- **THEN** 終端螢幕清除，顯示新的 prompt
