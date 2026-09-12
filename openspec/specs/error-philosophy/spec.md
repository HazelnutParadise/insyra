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

### Requirement: A constructor given input it cannot use reports it and returns nil

繪圖套件的每一個 `CreateXxx` SHALL NOT panic，無論設定結構是零值、必填欄位為空、資料列長度不一致、數量參數為負，或資料清單元素為 nil。無法繪製時 SHALL 記錄原因並回傳 nil；仍可繪製時 SHALL 回傳可用的圖表。

#### Scenario: A zero-value bar chart config
- **WHEN** `gplot.CreateBarChart(BarChartConfig{}, []float64{1, 2})`
- **THEN** 回傳可存檔的圖表，不 panic

#### Scenario: A ragged heat map grid
- **WHEN** `gplot.CreateHeatmapChart(cfg, [][]float64{{1,2,3},{4}})`
- **THEN** 記錄哪一列長度不同並回傳 nil，不 panic

#### Scenario: A nil data list
- **WHEN** `plot.CreateBarChart(cfg, nil)`
- **THEN** 回傳 nil，不 panic

### Requirement: A parser given malformed input skips what it cannot read

`lpgen` 的 LINGO 解析器 SHALL NOT panic，無論括號順序顛倒、缺少括號或宣告殘缺。讀不懂的宣告 SHALL 被略過，其餘內容 SHALL 照常解析。

#### Scenario: Reversed parentheses
- **WHEN** 模型中含有 `@BIN)X(;`
- **THEN** 該行被略過，模型的其他部分照常回傳

### Requirement: A save path is checked before it reaches a dependency

`plot.SavePNG` SHALL 在呼叫快照套件之前檢查輸出路徑帶有副檔名，缺少時 SHALL 回傳指出問題的錯誤。

#### Scenario: A path with no extension
- **WHEN** `plot.SavePNG(chart, "out")`
- **THEN** 回傳說明缺少副檔名的錯誤，不 panic
