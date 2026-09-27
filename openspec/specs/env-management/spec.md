# env-management Specification

## Purpose
TBD - created by archiving change cli-repl. Update Purpose after archive.

## Requirements

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
系統 SHALL 將環境狀態（變數快照）以帶型別的 JSON 格式存入 `state.json`，使變數在新的程序中還原後與儲存前相同：DataTable 保留欄位順序、欄名、列名、表名，以及每一格的 Go 型別與值；DataList 保留名稱與每一格的 Go 型別與值；頂層的值與值的切片保留 Go 型別；已擬合的 scaler 與階層分群樹保留完整狀態，還原後能直接交給後續命令使用。可保存的格子型別為所有內建整數與浮點數型別、`bool`、`string`、`time.Time`、`time.Duration`、`[]byte`、`decimal.Decimal`、`json.Number`，以及 `ReadJSON` 讀進來的巢狀 JSON 物件與陣列。NaN 與 ±Inf 照值保存，不是合法 UTF-8 的字串逐位元組保存，`time.Time` 保留時間點。先前版本寫入的 `state.json` SHALL 仍可還原，下次儲存時改以新格式寫入。

#### Scenario: Save DataTable variable
- **WHEN** 環境中有一個欄位依序為 `zeta`、`alpha`、`when` 的 DataTable 變數 `t1`，`zeta` 是 `float64`（含 `3.0` 這種整數值與 NaN），`alpha` 是 `string`，`when` 是 `time.Time`，環境儲存後在新的程序中還原
- **THEN** `t1` 的欄位順序仍是 `zeta`、`alpha`、`when`，每一格的 Go 型別與值、`nil` 格、列名與表名都與儲存前相同

#### Scenario: Save DataList variable
- **WHEN** 環境中有一個 DataList 變數 `d1`，混有 `int`、`int64`、`float64`、`bool`、`string`、`time.Time`、`nil` 與 NaN，環境儲存後還原
- **THEN** `d1` 的名稱與每一格的 Go 型別與值都與儲存前相同

#### Scenario: A date-difference column survives
- **WHEN** 一次性命令以 CCL 的日期相減（例如 `addcolccl d2 gap "B - A"`）加出一欄 `time.Duration`，下一個命令還原環境
- **THEN** 該表仍在，`gap` 欄每一格仍是 `time.Duration` 且值相同

#### Scenario: A fitted scaler is used by the next command
- **WHEN** 一次性命令 `scale fit std sc t cols zeta` 之後，另一個一次性命令執行 `scale transform sc t as t3`
- **THEN** 第二個命令成功，`t3` 與在同一個程序中擬合後直接轉換的結果逐位元相同

#### Scenario: A hierarchical tree is used by the next command
- **WHEN** 一次性命令 `hclust h complete as tr` 之後，另一個一次性命令執行 `cutree tr k 2 as lab`
- **THEN** 第二個命令成功，`lab` 與在同一個程序中切出的分群相同

#### Scenario: Restore variables
- **WHEN** 環境被載入
- **THEN** 系統從 `state.json` 反序列化所有變數，重建為儲存前的 Go 型別

#### Scenario: Read a state file written by an earlier release
- **WHEN** 環境的 `state.json` 由先前版本寫入，DataTable 以 JSON 字串或欄位陣列儲存，純量以 JSON 值儲存
- **THEN** 系統依先前版本的規則還原其中的變數，下次儲存時以新格式寫入

#### Scenario: A stored variable this build cannot decode
- **WHEN** `state.json` 中有一個新格式的變數無法解碼（例如較新版本寫入的格子型別）
- **THEN** 其他變數照常還原，該變數下次儲存時原樣寫回，kind 與名稱不變

#### Scenario: Export and import keep stored values
- **WHEN** 含 NaN 純量與大於 2^53 的 `int64` 的環境執行 `env export` 後再 `env import` 到另一個環境
- **THEN** 匯出成功，匯入後還原的值與型別與原環境相同

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

### Requirement: Variables the environment cannot store are reported when saving
系統儲存環境時 SHALL 寫入所有能保存的變數。無法保存的變數（例如迴歸結果）SHALL 不寫入 `state.json`。`env.Manager.SaveVariables` SHALL 回傳這些變數的名稱、Go 型別與原因；`SaveState` SHALL 維持原本的契約，只在檔案沒寫成時回傳錯誤。CLI 的一次性命令、REPL、腳本與 DSL session SHALL 在儲存當下為每個這種變數輸出一行警告，指出名稱、Go 型別、原因，以及它在程序結束或開啟另一個環境後就不存在。在同一個 REPL、腳本或 DSL session 中，同一個變數只要型別不變，SHALL 只警告一次。無法保存的變數 SHALL NOT 讓產生它的命令失敗，也 SHALL NOT 妨礙其他變數的保存。

#### Scenario: One-shot command stores a regression result
- **WHEN** 以一次性命令執行 `regression linear y x as r`
- **THEN** 命令成功，輸出一行指出 `r` 沒有存入環境的警告，其他變數照常保存，下一個命令看不到 `r`

#### Scenario: A table holding a value that is not a cell type
- **WHEN** 儲存的 DataTable 有一欄含有不在可保存清單中的值
- **THEN** 該表不寫入，警告指出是哪一欄、哪一種 Go 型別

#### Scenario: A session keeps an unsaved variable
- **WHEN** 在 REPL 或 DSL session 中建立一個無法保存的變數，之後再執行多個命令
- **THEN** 警告只出現一次，該變數在 session 結束或以 `env open` 開啟另一個環境之前都能使用
