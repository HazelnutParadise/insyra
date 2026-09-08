# datalist-parse-dates Specification

## Purpose
`DataList.ParseDates`／`DataTable.ParseDatesCols` 的字串轉 `time.Time` 規則（共用 ISO layout、不可解析變 nil、SQL 路徑共用）與 CLI `parsedates` 命令。

## Requirements
### Requirement: ParseDates on DataList and DataTable

`DataList` SHALL 提供 `ParseDates(layouts ...string) *DataList`：字串格子依序嘗試各 layout（未給時使用與 `ReadSQLOptions.ParseDates` 相同的預設清單），成功者轉為 UTC `time.Time`；已是 `time.Time` 的格子原樣保留；其他格子轉為 nil。`DataTable` SHALL 提供 `ParseDatesCols(cols []string, layouts ...string) *DataTable`，就地對指定欄位套用，欄位可用名稱或 Excel 索引，找不到的欄位 SHALL 發 warning 並略過。`ReadSQL` 的 `ParseDates` 選項 SHALL 改為呼叫同一個方法。兩者 SHALL 加入 `IDataList`／`IDataTable`。

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

- **WHEN** 既有 `ReadSQL … ParseDates` 測試執行
- **THEN** 結果與變更前相同

### Requirement: parsedates CLI command

CLI SHALL 提供 `parsedates <var> [cols c1,c2] [layout <go-layout>] [as <var>]`。`<var>` 為 `DataList` 時整列轉換；為 `DataTable` 時 `cols` 為必填；`layout` 可重複給。結果存到 `as` 或 `$result`。缺 `cols`、未知選項、非 DataList／DataTable 變數 SHALL 回傳含 `parsedates:` 前綴的錯誤。

#### Scenario: CSV to monthly bars end to end

- **WHEN** `.isr` 依序執行 `load bars.csv as dt`、`parsedates dt cols Date as dt`、`resample dt Date monthly Close:last as m`
- **THEN** `m` 有月線列且無錯誤

#### Scenario: Table without cols

- **WHEN** 執行 `parsedates dt`
- **THEN** 回傳指出 `cols` 必填的錯誤

