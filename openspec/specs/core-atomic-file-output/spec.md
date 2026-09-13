# core-atomic-file-output Specification

## Purpose
`ToCSV` 的寫入錯誤契約：包含 CSV writer 只在最後 flush 才回報的錯誤，一律回傳給呼叫端，不回報假成功。

## Requirements
### Requirement: ToCSV reports every write error

`ToCSV` 的任何寫入錯誤 SHALL 回傳，包含 CSV writer 只在最後 flush 時才回報的錯誤。

#### Scenario: Write failure
- **WHEN** 底層 writer 回傳錯誤
- **THEN** `ToCSV` 回傳該錯誤
