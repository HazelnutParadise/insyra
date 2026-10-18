## MODIFIED Requirements

### Requirement: Load data from file
系統 SHALL 提供 `load` 命令從檔案載入資料為 DataTable。

#### Scenario: Load CSV
- **WHEN** 使用者執行 `load "data.csv" as t1`
- **THEN** 系統呼叫 `ReadCSVFile()` 載入 CSV 並將結果存為變數 `t1`

#### Scenario: Load Excel with sheet
- **WHEN** 使用者執行 `load "data.xlsx" sheet Sales as t1`
- **THEN** 系統呼叫 `ReadExcelSheet()` 載入指定工作表

#### Scenario: Load JSON
- **WHEN** 使用者執行 `load "data.json" as t1`
- **THEN** 系統呼叫 `ReadJSONFile()` 載入 JSON

#### Scenario: Load Parquet
- **WHEN** 使用者執行 `load parquet "data.parquet" as t1`
- **THEN** 系統呼叫 parquet.Read() 載入 Parquet 檔

#### Scenario: Load without alias
- **WHEN** 使用者執行 `load "data.csv"`（無 `as`）
- **THEN** 結果存入 `$result`

#### Scenario: Auto format detection
- **WHEN** 使用者執行 `load "file.csv"` 或 `load "file.json"`
- **THEN** 系統依副檔名自動判斷格式

### Requirement: Time series commands
系統 SHALL 提供時間序列命令：`movavg`、`expsmooth`、`diff`、`fillnan`。

#### Scenario: Moving average
- **WHEN** 使用者執行 `movavg d1 5 as d1_ma`
- **THEN** 系統自行計算每個視窗的平均值（以 `insyra.ToFloat64Safe` 讀取格子），並存結果；輸出比輸入短

#### Scenario: Fill NaN with mean
- **WHEN** 使用者執行 `fillnan d1 mean`
- **THEN** 系統透過 `fillna` 機制把 NaN 填為平均值
