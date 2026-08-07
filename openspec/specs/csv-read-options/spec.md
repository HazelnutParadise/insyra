# csv-read-options Specification

## Purpose
根套件的 options 式 CSV 讀取 API，涵蓋列/欄名稱設定、編碼指定，以及跳過欄位型別推斷的 raw-strings 模式。既有的 `ReadCSV_File` / `ReadCSV_String` 維持原簽名，委派給 options 版本實作。

## Requirements
### Requirement: Options-based CSV loading API
根套件 SHALL 提供 `CSVReadOptions` 結構與 `ReadCSV_FileWithOptions(filePath string, opts CSVReadOptions)`、`ReadCSV_StringWithOptions(csvString string, opts CSVReadOptions)` 兩個函數。`CSVReadOptions` SHALL 包含 `FirstColToRowNames bool`、`FirstRowToColNames bool`、`Encoding string`（僅檔案輸入有效，空字串或 `"auto"` 表示自動偵測）與 `RawStrings bool`。零值 options SHALL 產生與現有 `ReadCSV_File(path, false, false)` / `ReadCSV_String(s, false, false)` 完全相同的結果。

#### Scenario: Zero-value options match legacy behavior
- **WHEN** 呼叫 `ReadCSV_StringWithOptions(csv, CSVReadOptions{})`
- **THEN** 結果與 `ReadCSV_String(csv, false, false)` 相同，欄位型別推斷照常執行

#### Scenario: Row/col name flags map to legacy parameters
- **WHEN** 呼叫 `ReadCSV_FileWithOptions(path, CSVReadOptions{FirstColToRowNames: true, FirstRowToColNames: true})`
- **THEN** 結果與 `ReadCSV_File(path, true, true)` 相同

### Requirement: Raw-strings mode disables type inference
當 `CSVReadOptions.RawStrings` 為 true 時，系統 SHALL 將每個 cell 保留為 CSV 解析後的原始字串，SHALL NOT 執行任何欄位型別推斷或數值轉換。

#### Scenario: Leading zeros preserved
- **WHEN** 以 `RawStrings: true` 載入含股票代號欄 `2330`、`0050`、`00878` 的 CSV
- **THEN** 三個 cell 皆為字串，`0050` 與 `00878` 的開頭零完整保留

#### Scenario: Exact amounts stay strings
- **WHEN** 以 `RawStrings: true` 載入含金額 `600.855` 與千分位 `"2,000"` 的 CSV
- **THEN** cell 值為字串 `"600.855"` 與 `"2,000"`，不轉為 float64

#### Scenario: Empty cell stays empty string
- **WHEN** 以 `RawStrings: true` 載入含空 cell 的 CSV
- **THEN** 該 cell 為空字串 `""`，不轉為 NaN 或 nil

### Requirement: Legacy CSV functions remain unchanged
`ReadCSV_File` 與 `ReadCSV_String` 的簽名與行為 SHALL 維持不變，並委派給 options 版本實作。

#### Scenario: Legacy inference still applies
- **WHEN** 呼叫 `ReadCSV_String(csv, false, true)` 載入全整數欄
- **THEN** 該欄仍推斷為 int64（與既有行為一致）

### Requirement: isr CSV input supports raw strings
`isr` 套件的 `CSV_inOpts` SHALL 提供 `RawStrings bool` 欄位，`DT.From(CSV{...})` SHALL 將其傳遞至 options 版讀取函數。

#### Scenario: isr passes RawStrings through
- **WHEN** 呼叫 `DT.From(CSV{String: csv, InputOpts: CSV_inOpts{FirstRow2ColNames: true, RawStrings: true}})`
- **THEN** 產生的 DataTable 所有 cell 皆為原始字串
