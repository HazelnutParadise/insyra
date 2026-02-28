## ADDED Requirements

### Requirement: Environment create
系統 SHALL 提供 `env create <name>` 命令建立新環境。

#### Scenario: Create new environment
- **WHEN** 使用者執行 `env create my-project`
- **THEN** 系統在 `~/.insyra/envs/my-project/` 建立目錄，初始化空的 `state.json`、`history.txt`、`config.json`

#### Scenario: Create duplicate environment
- **WHEN** 使用者執行 `env create my-project` 但該環境已存在
- **THEN** 系統顯示錯誤訊息提示環境已存在

### Requirement: Environment list
系統 SHALL 提供 `env list` 命令列出所有環境。

#### Scenario: List environments
- **WHEN** 使用者執行 `env list`
- **THEN** 系統列出所有環境名稱，標示當前環境和最後存取時間

#### Scenario: No environments exist
- **WHEN** 使用者執行 `env list` 但無任何環境
- **THEN** 系統顯示提示訊息建議建立第一個環境

### Requirement: Environment open
系統 SHALL 提供 `env open <name>` 命令進入指定環境的 REPL。

#### Scenario: Open existing environment
- **WHEN** 使用者執行 `env open my-project`
- **THEN** 系統載入該環境的狀態並進入 REPL，prompt 顯示 `insyra [my-project] > `

#### Scenario: Open non-existent environment
- **WHEN** 使用者執行 `env open nonexistent`
- **THEN** 系統顯示錯誤訊息提示環境不存在

### Requirement: Environment delete
系統 SHALL 提供 `env delete <name>` 命令刪除環境。

#### Scenario: Delete with confirmation
- **WHEN** 使用者執行 `env delete my-project`
- **THEN** 系統顯示確認提示，使用者確認後刪除 `~/.insyra/envs/my-project/` 目錄

#### Scenario: Delete current environment
- **WHEN** 使用者在 REPL 中嘗試刪除當前使用的環境
- **THEN** 系統顯示錯誤訊息，禁止刪除當前環境

### Requirement: Environment rename
系統 SHALL 提供 `env rename <old> <new>` 命令重命名環境。

#### Scenario: Rename environment
- **WHEN** 使用者執行 `env rename old-name new-name`
- **THEN** 系統將 `~/.insyra/envs/old-name/` 重命名為 `~/.insyra/envs/new-name/`

### Requirement: Environment info
系統 SHALL 提供 `env info` 命令顯示當前環境資訊。

#### Scenario: Show environment info
- **WHEN** 使用者執行 `env info`
- **THEN** 系統顯示當前環境名稱、路徑、變數數量、最後存取時間

### Requirement: State serialization
系統 SHALL 將環境狀態（變數快照）序列化為 JSON 格式存入 `state.json`。

#### Scenario: Save DataTable variable
- **WHEN** 環境中有一個 DataTable 變數 `t1`，環境被儲存
- **THEN** `state.json` 中包含 `t1` 的完整資料（使用 `ToJSON_String()` 序列化）、欄名、列名和型別標記

#### Scenario: Save DataList variable
- **WHEN** 環境中有一個 DataList 變數 `d1`，環境被儲存
- **THEN** `state.json` 中包含 `d1` 的資料陣列和名稱

#### Scenario: Restore variables
- **WHEN** 環境被載入
- **THEN** 系統從 `state.json` 反序列化所有變數，重建為對應的 DataTable / DataList 物件

### Requirement: Command history persistence
系統 SHALL 將命令歷史持久化存入 `history.txt`。

#### Scenario: Save history
- **WHEN** 使用者在 REPL 中輸入命令
- **THEN** 命令被追加到 `history.txt`

#### Scenario: History command
- **WHEN** 使用者執行 `history`
- **THEN** 系統顯示該環境的命令歷史記錄

### Requirement: Default environment auto-creation
系統 SHALL 在首次使用時自動建立 `default` 環境。

#### Scenario: First run
- **WHEN** 使用者首次執行 `insyra`，`~/.insyra/` 不存在
- **THEN** 系統自動建立 `~/.insyra/envs/default/` 並進入 REPL

### Requirement: Global config
系統 SHALL 在 `~/.insyra/config.json` 存放全域設定。

#### Scenario: Config command
- **WHEN** 使用者執行 `config`
- **THEN** 系統顯示當前全域設定（預設環境、log level 等）

#### Scenario: Set config value
- **WHEN** 使用者執行 `config log-level debug`
- **THEN** 系統更新全域設定中的 log-level 為 debug
