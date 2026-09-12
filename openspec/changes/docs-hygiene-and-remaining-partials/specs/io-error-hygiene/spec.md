## ADDED Requirements

### Requirement: A named sheet that is not there is an error

`ExcelToCsv` 收到 `onlyContainSheets` 時，其中不存在於檔案的名稱 SHALL 回報錯誤並列出檔案實際有的工作表，SHALL NOT 靜默略過。

#### Scenario: A misspelled sheet name
- **WHEN** `onlyContainSheets` 含一個檔案裡沒有的名稱
- **THEN** 回傳錯誤，指出缺少哪些、檔案有哪些
