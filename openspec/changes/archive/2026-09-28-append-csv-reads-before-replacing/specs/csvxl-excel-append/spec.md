## ADDED Requirements

### Requirement: A CSV that cannot be read leaves its sheet untouched

`csvxl.AppendCsvToExcel` SHALL read each CSV in full before it creates or replaces that CSV's sheet. When a CSV cannot be opened, its encoding cannot be detected, or its content is not valid CSV, the sheet it targets SHALL keep its content and settings, or SHALL NOT be created when the workbook had no such sheet, the other CSVs of the call SHALL still be appended, and the call SHALL return the `<n> files failed to append` error it returns today.

#### Scenario: A missing CSV does not empty the sheet

- **WHEN** 工作表 `data` 原有 3 列，對 `data` 執行 `AppendCsvToExcel`，指定的 CSV 不存在
- **THEN** 呼叫回傳 `1 files failed to append`，重新開啟後 `data` 仍是原本的 3 列

#### Scenario: Invalid CSV does not empty the sheet

- **WHEN** 工作表 `data` 原有 3 列，對 `data` 執行 `AppendCsvToExcel`，CSV 內容有未閉合的引號
- **THEN** 呼叫回傳 `1 files failed to append`，重新開啟後 `data` 仍是原本的 3 列

#### Scenario: One bad CSV in a batch

- **WHEN** 工作表 `good` 與 `bad` 各有內容，一次 `AppendCsvToExcel` 兩個 CSV，給 `good` 的可讀、給 `bad` 的不存在
- **THEN** 呼叫回傳 `1 files failed to append`，`good` 只有新 CSV 的內容，`bad` 保留原本的內容

#### Scenario: No empty sheet for a CSV that cannot be read

- **WHEN** 工作簿沒有 `extra` 工作表，對 `extra` 執行 `AppendCsvToExcel`，指定的 CSV 不存在
- **THEN** 呼叫回傳 `1 files failed to append`，重新開啟後工作表清單不變
