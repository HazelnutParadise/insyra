# core-config-and-error-hook-safety Specification

## Purpose
`Config` 全域設定與錯誤 hook 的並行安全：設定器與熱路徑讀取不 race、hook 依序且有上限、匯入套件不印任何東西。

## Requirements
### Requirement: Config fields are safe to set concurrently

`Config.SetLogLevel`、`SetUseColoredOutput`、`SetDontPanic`、`SetDefaultErrHandlingFunc` 與對應的讀取 SHALL 是原子操作，在 `go test -race` 下與 `LogInfo`／`LogWarning` 並行呼叫 SHALL NOT 回報 data race。

#### Scenario: Concurrent set and log
- **WHEN** 一個 goroutine 反覆 `SetLogLevel`，另一個反覆 `LogInfo`
- **THEN** race detector 無回報

### Requirement: The error hook is bounded and ordered

透過 `SetDefaultErrHandlingFunc` 設定的 hook SHALL 由單一 goroutine 依產生順序呼叫；佇列有上限，超過時 SHALL 丟棄 hook 呼叫但仍寫入錯誤環。

#### Scenario: Hook sees errors in order
- **WHEN** 連續發出 100 個 warning
- **THEN** hook 依序收到這 100 個訊息

### Requirement: Importing the library prints nothing

`import "github.com/HazelnutParadise/insyra"` SHALL NOT 在 stdout／log 輸出任何橫幅；CLI REPL 啟動時 SHALL 自行印出。

#### Scenario: Library import
- **WHEN** 程式只 import 核心套件
- **THEN** 沒有 "Welcome to Insyra" 輸出

