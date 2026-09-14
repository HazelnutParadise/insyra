# error-philosophy Specification

## Purpose
在這條版本線上，下列路徑不得結束宿主程序或 panic：`isr` 包裝器、`gplot` 與 `plot` 的圖表建構（包括收到無法使用的輸入時）、`plot.SavePNG` 的輸出路徑檢查、`lpgen` 的 LINGO 解析、`lp` 的 GLPK 安裝與暫存檔、`py` 的 IPC 監聽，失敗時記錄警告後返回。v0.3.2 在 `Config.SetDontPanic(true)` 下能正常返回的呼叫，不論設定為何都回傳當時的值。

## Requirements
### Requirement: isr wrappers log and return the v0.3.2 values instead of exiting

`isr` 的 `DT.From`、`Col`、`Row`、`Push`、`UseDL`、`UseDT` 遇到不支援的型別或讀檔失敗時 SHALL NOT 結束程序，SHALL 記錄警告，並不論 `SetDontPanic` 設定 SHALL 回傳 v0.3.2 在 `SetDontPanic(true)` 下的值：讀不到來源或型別不支援時包著 nil `DataTable`，選擇器型別不支援時包著 nil `DataList`，加不進去的 `Row`／`Col` 被略過、其餘照常加入。回傳的表格不是 nil 時 SHALL 在其 `Err()` 記錄錯誤。

#### Scenario: Missing file
- **WHEN** 在 `SetDontPanic` 開或關時呼叫 `isr.DT.From(isr.CSV{FilePath: "no_such.csv"})`
- **THEN** 回傳非 nil 的 `*dt`，其 `DataTable` 為 nil，程序繼續

#### Scenario: A row that cannot be added
- **WHEN** `isr.DT.From(isr.Rows{{"A": 1}, {"A": 1, 0: 2}, {"A": 3}})`
- **THEN** 回傳 2 列的表格，其 `Err()` 非 nil

### Requirement: A radar chart without indicators is still returned

`plot.CreateRadarChart` 收到序列、但 `Indicators` 與 `MaxValues` 都沒有設定時 SHALL 記錄警告，SHALL NOT 結束程序，並不論 `SetDontPanic` 設定 SHALL 回傳沒有 indicators 的圖表，與 v0.3.2 在 `SetDontPanic(true)` 下相同。

#### Scenario: No indicators
- **WHEN** `CreateRadarChart(RadarChartConfig{}, []RadarSeries{{Name: "s", Values: []float32{1, 2, 3}}})`
- **THEN** 回傳可存成 HTML 的非 nil 圖表

### Requirement: Chart, solver and IPC failures log a warning

`gplot.CreateHistogram`（`Bins` 不大於 0 時採用 10）、`gplot.CreateLineChart`／`CreateStepChart` 無法建立序列、`plot.CreateHeatMap` 日曆模式設定錯誤、`lp` 的 GLPK 下載／解壓／編譯與 `SolveModel` 暫存檔失敗、`py` 的 IPC 監聽失敗 SHALL 記錄警告後返回，SHALL NOT 結束程序或 panic。

#### Scenario: Calendar heatmap misuse
- **WHEN** 以 `UseCalendar: true` 與 int 型別的 X 呼叫 `CreateHeatMap`
- **THEN** 回傳 nil 且不 panic

### Requirement: A chart constructor given input it cannot use reports it and returns nil

`gplot.CreateBarChart`（沒有 `XAxis`）、`gplot.CreateFunctionPlot`（nil 函式）、`gplot.CreateHeatmapChart`（某列比第 0 列短，或 `Colors` 為負）、`plot.CreateBarChart`／`CreateLineChart`／`CreateBoxPlot`（含 nil 的 `IDataList`）與 `plot.CreateWordCloud`（nil 清單）SHALL NOT panic。無法繪製時 SHALL 記錄警告並回傳 nil；仍可繪製時 SHALL 回傳可用的圖表，比第 0 列長的列照常繪製。

#### Scenario: A zero-value bar chart config
- **WHEN** `gplot.CreateBarChart(BarChartConfig{}, []float64{1, 2})`
- **THEN** 回傳可存檔的圖表，不 panic

#### Scenario: A heat map row shorter than row 0
- **WHEN** `gplot.CreateHeatmapChart(cfg, [][]float64{{1,2,3},{4}})`
- **THEN** 記錄哪一列比第 0 列短並回傳 nil，不 panic

#### Scenario: A heat map row longer than row 0
- **WHEN** `gplot.CreateHeatmapChart(cfg, [][]float64{{1},{2,3}})`
- **THEN** 回傳可存檔的圖表，多出來的值被忽略

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
