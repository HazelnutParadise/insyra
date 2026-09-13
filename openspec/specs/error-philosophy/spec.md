# error-philosophy Specification

## Purpose
在這條版本線上，下列失敗路徑不得結束宿主程序或 panic：`isr` 包裝器、`gplot` 與 `plot` 的圖表建構、`lp` 的 GLPK 安裝與暫存檔、`py` 的 IPC 監聽，失敗時記錄後返回。

## Requirements
### Requirement: isr wrappers record instead of exiting

`isr` 的 `DT.From`、`Col`、`Row`、`Push`、`UseDL`、`UseDT` 遇到不支援的型別或讀檔失敗時 SHALL NOT 結束程序，SHALL 回傳可繼續使用的物件並在其上記錄 `Err()`。

#### Scenario: Missing file
- **WHEN** `isr.DT.From(isr.CSV{FilePath: "no_such.csv"})`
- **THEN** 回傳非 nil 的 `*dt`，其 `Err()` 非 nil，程序繼續

### Requirement: Chart, solver and IPC failures log a warning

`gplot.CreateHistogram`（`Bins` 不大於 0 時採用 10）、`gplot.CreateLineChart`／`CreateStepChart` 無法建立序列、`plot.CreateRadarChart` 缺少 indicators、`plot.CreateHeatMap` 日曆模式設定錯誤、`lp` 的 GLPK 下載／解壓／編譯與 `SolveModel` 暫存檔失敗、`py` 的 IPC 監聽失敗 SHALL 記錄警告後返回，SHALL NOT 結束程序或 panic。

#### Scenario: Calendar heatmap misuse
- **WHEN** 以 `UseCalendar: true` 與 int 型別的 X 呼叫 `CreateHeatMap`
- **THEN** 回傳 nil 且不 panic
