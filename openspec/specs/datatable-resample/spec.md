# datatable-resample Specification

## Purpose
`DataTable` 的日曆週期重取樣：以 `time.Time` 欄位分週／月／季／年，重用 `GroupBy` 的 `AggregateOp` 做 OHLCV 等聚合，每個非空期間一列並以期間最後一個日曆日標籤。

## Requirements
### Requirement: Time-based resampling of a DataTable

`DataTable` SHALL 提供 `Resample(timeCol string, freq ResampleFreq, aggs ...ResampleAgg) (*DataTable, error)`。`ResampleFreq` SHALL 為 `ResampleWeekly`（週一到週日）、`ResampleMonthly`、`ResampleQuarterly`、`ResampleYearly`。`ResampleAgg{Col string; Op AggregateOp; As string}` SHALL 重用 `GroupBy` 的 `AggregateOp`；`As` 為空時輸出欄名為 `Col`。輸出 SHALL 每個非空期間一列，依期間遞增排序，`timeCol` 欄位值為該期間最後一個日曆日（`time.Time`，時區沿用輸入值），空期間不補列。`timeCol` 中任一非 `time.Time` 值、找不到的欄位、空的 `aggs`、未知的 `freq` SHALL 回傳錯誤。輸入列的原始順序不影響結果。

#### Scenario: Daily OHLCV to monthly bars

- **WHEN** 輸入為兩個月的日線（Date, Open, High, Low, Close, Volume），以 `ResampleMonthly` 與 `OpFirst`/`OpMax`/`OpMin`/`OpLast`/`OpSum` 呼叫
- **THEN** 輸出兩列，`Date` 為各月最後一個日曆日，`Open` 為該月第一筆、`Close` 為最後一筆、`High`/`Low` 為極值、`Volume` 為總和

#### Scenario: Weekly buckets run Monday to Sunday

- **WHEN** 輸入含週五與下週一的兩筆
- **THEN** 輸出兩列，第一列 `Date` 為該週日，第二列為下一個週日

#### Scenario: Empty periods are not fabricated

- **WHEN** 輸入的日期跳過整個三月
- **THEN** 輸出沒有三月那一列

#### Scenario: Non-time cell is refused

- **WHEN** `timeCol` 含一個字串
- **THEN** 回傳指出列號的錯誤，不產生輸出

#### Scenario: Output column naming

- **WHEN** `ResampleAgg{Col: "Close", Op: OpLast, As: "MonthClose"}`
- **THEN** 輸出欄名為 `MonthClose`；未指定 `As` 的欄位沿用原欄名

### Requirement: Resample appears on the interface

`IDataTable` SHALL 宣告 `Resample(timeCol string, freq ResampleFreq, aggs ...ResampleAgg) (*DataTable, error)`。

#### Scenario: Interface satisfaction compiles

- **WHEN** 套件編譯
- **THEN** `*DataTable` 仍滿足 `IDataTable`

