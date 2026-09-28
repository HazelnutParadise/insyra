## RENAMED Requirements

- FROM: `### Requirement: A constructor given input it cannot use reports it and returns nil`
- TO: `### Requirement: A constructor given input it cannot use returns an error`

## MODIFIED Requirements

### Requirement: A constructor given input it cannot use returns an error

繪圖套件的每一個 `CreateXxx` SHALL NOT panic，無論設定結構是零值、必填欄位為空、資料表是空的、數量參數為負，或資料清單元素為 nil。無法繪製時 SHALL 回傳 nil 圖表與說明原因的錯誤；仍可繪製時 SHALL 回傳可用的圖表與 nil 錯誤。

#### Scenario: A zero-value bar chart config
- **WHEN** `gplot.CreateBarChart(BarChartConfig{}, insyra.NewDataList(1, 2))`
- **THEN** 回傳可存檔的圖表與 nil 錯誤，不 panic

#### Scenario: A ragged heat map grid
- **WHEN** `gplot.CreateHeatmapChart(cfg, insyra.NewDataTable(insyra.NewDataList(1.0, 2.0), insyra.NewDataList(3.0)))`，兩欄長度不同
- **THEN** 回傳圖表與 nil 錯誤，較短一欄補上的空格畫成 0，不 panic

#### Scenario: An empty table for a heat map
- **WHEN** `gplot.CreateHeatmapChart(cfg, insyra.NewDataTable())`
- **THEN** 回傳 nil 圖表與錯誤，不 panic

#### Scenario: A nil data list
- **WHEN** `plot.CreateBarChart(cfg, nil)`
- **THEN** 回傳 nil 圖表與錯誤，不 panic
