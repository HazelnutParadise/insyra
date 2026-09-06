## ADDED Requirements

### Requirement: ToCSV and ToJSON are atomic and honest

`ToCSV`／`ToJSON` SHALL 先寫入同目錄暫存檔再 rename；任何寫入或 flush 錯誤 SHALL 回傳；失敗時 SHALL NOT 留下目標檔或暫存檔。

#### Scenario: Write failure
- **WHEN** 底層 writer 回傳錯誤
- **THEN** `ToCSV` 回傳該錯誤

#### Scenario: Missing directory
- **WHEN** 目標目錄不存在
- **THEN** 回傳錯誤且目錄樹不新增任何檔案
