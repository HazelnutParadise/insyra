## MODIFIED Requirements

### Requirement: Load data from file
系統 SHALL 提供 `load` 命令從檔案載入資料為 DataTable。

#### Scenario: Load CSV
- **WHEN** 使用者執行 `load "data.csv" as t1`
- **THEN** 系統呼叫 `ReadCSVFile()` 載入 CSV 並將結果存為變數 `t1`

#### Scenario: Load Excel with sheet
- **WHEN** 使用者執行 `load "data.xlsx" sheet Sales as t1`
- **THEN** 系統呼叫 `excel.Read()` 載入指定工作表

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
