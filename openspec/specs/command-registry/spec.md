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

### Requirement: Acceleration handler registration
The command registry SHALL register acceleration-related handlers so CLI and REPL entry points can share the same control surface.

#### Scenario: Registry dispatches accel handler
- **WHEN** an accel command is dispatched through `Registry.Dispatch`
- **THEN** the registry routes the request to the accel handler with the shared execution context

### Requirement: Acceleration execution report visibility
The acceleration handler SHALL expose selected backend, selected devices, and fallback outcome through the shared execution path.

#### Scenario: Accel-enabled command completes
- **WHEN** acceleration-enabled execution finishes
- **THEN** the handler can surface backend choice, selected devices, and fallback reason through the shared execution path

### Requirement: The registry is read through a locked accessor

讀取 `Registry` SHALL 經過取用 `registryMu` 的存取函式（`LookupCommand`、`SnapshotRegistry`），SHALL NOT 直接走訪 map。嵌入者可以在任何時候註冊指令。

#### Scenario: help and tab completion
- **WHEN** `help` 或 REPL 的補全列出指令
- **THEN** 它們透過加鎖的快照讀取，而不是直接走訪 `Registry`

### Requirement: The help table lines up

`help` 的指令清單 SHALL 依最長的指令名稱決定欄寬，SHALL NOT 使用固定寬度。

#### Scenario: A command name longer than twelve characters
- **WHEN** 清單中含 `knn_neighbors`
- **THEN** 所有描述都從同一欄開始

### Requirement: Commands declare their one-shot flags at registration

A `CommandHandler` SHALL declare the flags its one-shot form takes in `Flags`. `BuildCobraCommands` SHALL register each declared flag with Cobra under its name and help text, and SHALL append each one that is set to the arguments `Run` receives, in declared order: a switch as `--name`, a flag that takes a value as `--name value` when the value is not blank. A flag with a `Form` SHALL be appended only when the first argument is that word, in any letter case. `BuildCobraCommands` SHALL NOT single out a command by name to add or forward a flag.

`Register` SHALL refuse a handler with a flag that has no name or a name declared twice, with an error and without registering it.

#### Scenario: A flag declared twice
- **WHEN** a handler declares `--force` twice and is registered
- **THEN** `Register` returns an error and the command is not registered

#### Scenario: env clear keeps its history flag
- **WHEN** the shell runs `insyra env clear a --keep-history`
- **THEN** `env`'s Run receives `clear a --keep-history`

#### Scenario: A flag outside its form is dropped
- **WHEN** the shell runs `insyra env clear a --force`
- **THEN** `env`'s Run receives `clear a`

#### Scenario: A flag with a value
- **WHEN** the shell runs `insyra accel devices --mode cpu`
- **THEN** `accel`'s Run receives `devices --mode cpu`

### Requirement: accel reads its mode from the session's manager

When `accel` is run without `--mode`, it SHALL read the default mode from the global config of the session's environment manager (`ExecContext.Env`), and from `Default()` only when the session has none.

#### Scenario: A session with its own manager
- **WHEN** `Default()`'s config sets `accel-mode` to `gpu`, the session's manager sets it to `cpu`, and `accel` runs without `--mode`
- **THEN** the mode is `cpu`

### Requirement: A command answers to its aliases everywhere
`Dispatch` 與 `LookupCommand` SHALL 以指令的 `Name` 或任一 `Aliases` 找到同一個指令，在 one-shot、REPL、腳本、Go session 與 `help <alias>` 都一樣。`Register` SHALL 拒絕名稱是其他指令別名的 handler，以及任一別名已經是某個指令名稱或別名的 handler，並回傳錯誤且不註冊，讓同一個字永遠只對應一個指令。

#### Scenario: help names a command by its alias
- **WHEN** 使用者執行 `help quit`
- **THEN** 輸出 `exit` 的說明，而不是 unknown command

#### Scenario: An alias already taken
- **WHEN** 程式註冊一個指令，它的別名是另一個已註冊指令的別名或名稱
- **THEN** `Register` 回傳錯誤，該指令沒有被註冊

### Requirement: A one-shot flag may belong to several forms
`CommandFlag.Form` MAY 以 `|` 分隔列出多個字（例如 `"import|delete"`），寫法與 Usage 的選項相同；第一個參數等於其中任一個字（不分大小寫）時，`BuildCobraCommands` SHALL 把這個旗標交給 `Run`，否則 SHALL 丟掉它。

#### Scenario: --force reaches env delete
- **WHEN** the shell runs `insyra --env work env delete default --force`
- **THEN** `env`'s Run receives `delete default --force`

#### Scenario: --force is still dropped from env clear
- **WHEN** the shell runs `insyra env clear a --force`
- **THEN** `env`'s Run receives `clear a`

