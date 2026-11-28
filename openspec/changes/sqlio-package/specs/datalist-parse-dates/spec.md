## MODIFIED Requirements

### Requirement: ParseDates on DataList and DataTable

`DataList` SHALL 提供 `ParseDates(layouts ...string) *DataList`：字串格子依序嘗試各 layout（未給時使用與 `sqlio.ReadOptions.ParseDates` 相同的預設清單），成功者轉為 UTC `time.Time`；已是 `time.Time` 的格子原樣保留；其他格子轉為 nil。`DataTable` SHALL 提供 `ParseDatesCols(cols []string, layouts ...string) *DataTable`，就地對指定欄位套用，欄位可用名稱或 Excel 索引，找不到的欄位 SHALL 發 warning 並略過。`sqlio` 的 `ParseDates` 選項 SHALL 改為呼叫同一個方法。兩者 SHALL 加入 `IDataList`／`IDataTable`。

#### Scenario: ISO strings become time.Time

- **WHEN** `NewDataList("2026-09-01", "2026-09-02", 3, "bad")` 呼叫 `ParseDates()`
- **THEN** 前兩格為對應日期的 UTC `time.Time`，後兩格為 nil

#### Scenario: Custom layout

- **WHEN** `ParseDates("2006/01/02")` 對 `"2026/09/01"`
- **THEN** 轉為 2026-09-01

#### Scenario: Table columns in place

- **WHEN** `ParseDatesCols([]string{"Date"})` 後執行 `Resample("Date", ResampleMonthly, …)`
- **THEN** `Resample` 不再拒絕該欄

#### Scenario: SQL path unchanged

- **WHEN** 既有 `sqlio.Read … ParseDates` 測試執行
- **THEN** 結果與變更前相同
