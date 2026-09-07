# error-philosophy Specification

## Purpose
錯誤哲學：程式庫預設不結束、不中斷宿主程序；要 fail-fast 由呼叫端明確開啟。

## Requirements
### Requirement: The library never terminates the host process

任何 insyra 套件在預設設定下 SHALL NOT 呼叫 `os.Exit`，SHALL NOT panic。`LogFatal` SHALL 記錄錯誤後返回。

#### Scenario: Fatal path with default config
- **WHEN** 以預設設定呼叫觸發 `LogFatal` 的程式路徑
- **THEN** 程序繼續執行，錯誤出現在全域緩衝區中

### Requirement: Panicking is opt-in

`Config.SetPanicOnError(true)` SHALL 使任何被記錄的錯誤（含 `LogFatal` 與寫入 `Err()` 的錯誤）立即 panic，panic 值 SHALL 實作 `error`，且 SHALL NOT 使用 `os.Exit`，呼叫端 SHALL 能 `recover`。

#### Scenario: Opt-in panic is recoverable
- **WHEN** 開啟 `SetPanicOnError(true)` 後觸發一個錯誤，呼叫端以 `recover()` 攔截
- **THEN** `recover()` 回傳的值可斷言為 `error`，程序不結束

### Requirement: Error is a distinct log level

`LogLevelError` SHALL 存在於 `LogLevelWarning` 與 `LogLevelFatal` 之間，`LogError` SHALL 以該等級記錄；寫入實例 `Err()` 的紀錄 SHALL 使用 `LogLevelError`。

#### Scenario: Level ordering
- **WHEN** 比較四個等級
- **THEN** `LogLevelDebug < LogLevelInfo < LogLevelWarning < LogLevelError < LogLevelFatal`

