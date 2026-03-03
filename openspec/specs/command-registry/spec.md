# command-registry Specification

## Purpose
TBD - created by archiving change cli-repl. Update Purpose after archive.
## Requirements
### Requirement: Unified command handler interface
系統 SHALL 定義統一的 `CommandHandler` 介面，所有命令邏輯通過此介面註冊，CLI、REPL 和腳本執行三個入口共用同一套 handler。

#### Scenario: Handler registration
- **WHEN** 系統啟動
- **THEN** 所有命令 handler 透過 Registry 註冊，每個 handler 具有 Name、Aliases、Usage、Description 和 Run 函式

#### Scenario: CLI routing
- **WHEN** 使用者透過 CLI 執行 `insyra show t1`
- **THEN** cobra 將 `show` 路由到 Registry 中對應的 handler，傳入 `["t1"]` 作為 args

#### Scenario: REPL routing
- **WHEN** 使用者在 REPL 中輸入 `show t1`
- **THEN** REPL 解析第一個 token `show`，查 Registry 路由到同一個 handler

#### Scenario: Script routing
- **WHEN** 腳本檔中包含一行 `show t1`
- **THEN** 腳本執行器逐行解析，同樣路由到 Registry 中的 handler

### Requirement: Shared execution context
系統 SHALL 維護一個 `ExecContext`，包含變數表（`map[string]any`）、環境名稱、環境路徑和輸出目標，在 handler 間共享。

#### Scenario: Variable sharing between commands
- **WHEN** 使用者執行 `load "data.csv" as t1` 後執行 `show t1`
- **THEN** `show` handler 從共享的變數表中取得 `t1` 並顯示

#### Scenario: Implicit result variable
- **WHEN** 使用者執行一條產生結果的命令但未使用 `as <var>`
- **THEN** 結果自動存入 `$result` 變數，可被後續命令引用

### Requirement: Stateful CLI commands
有狀態的 CLI 命令（需要讀寫變數表的命令）SHALL 自動載入當前環境的 state，執行後自動存回。

#### Scenario: CLI stateful execution
- **WHEN** 使用者執行 `insyra load "data.csv" as t1` 然後執行 `insyra show t1`
- **THEN** 第一條命令將 `t1` 存入環境 state，第二條命令從 state 讀取 `t1` 並顯示

#### Scenario: Stateless commands skip state
- **WHEN** 使用者執行 `insyra version` 或 `insyra help`
- **THEN** 系統不讀取或寫入環境 state

