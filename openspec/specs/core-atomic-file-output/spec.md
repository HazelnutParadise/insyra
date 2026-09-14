# core-atomic-file-output Specification

## Purpose
`ToCSV` 的寫入錯誤契約：包含 CSV writer 只在最後 flush 才回報的錯誤與關閉檔案時的錯誤，一律回傳給呼叫端，不回報假成功。

## Requirements
### Requirement: ToCSV reports every write error

`ToCSV`／`ToCSVWithOptions` 的任何寫入錯誤 SHALL 回傳，包含 CSV writer 只在最後 flush 時才回報的錯誤；寫入本身成功時，關閉檔案的錯誤 SHALL 回傳。

#### Scenario: Write failure
- **WHEN** 底層 writer 回傳錯誤
- **THEN** `ToCSV` 回傳該錯誤

#### Scenario: Close failure after a successful write
- **WHEN** 資料都寫入成功，但關閉檔案回傳錯誤
- **THEN** `ToCSV` 回傳關閉的錯誤
