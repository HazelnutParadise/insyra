## MODIFIED Requirements

### Requirement: Convert format
系統 SHALL 提供 `convert` 命令進行格式轉換。CSV 轉 Excel 時寫成一張工作表。Excel 轉 CSV 時 SHALL 以 `sheet <工作表> <檔名>` 成對指定，可重複多組，每組把該工作表寫成該檔，其他工作表不轉；`convert <file.xlsx> <file.csv>` 只在活頁簿只有一張工作表時轉換，否則 SHALL 回錯誤並列出工作表名稱。不存在的工作表、同一個檔名出現兩次，SHALL 在寫出任何檔案前回錯誤。

#### Scenario: CSV to Excel
- **WHEN** 使用者執行 `convert "data.csv" "data.xlsx"`
- **THEN** 系統轉換 CSV 為 Excel

#### Scenario: Pairs of sheets and files
- **WHEN** 活頁簿有 `Q1`、`Q2`、`Q3`，使用者執行 `convert book.xlsx sheet Q1 q1.csv sheet Q3 q3.csv`
- **THEN** 只寫出 `q1.csv`（Q1）與 `q3.csv`（Q3）

#### Scenario: One output file for a workbook of several sheets
- **WHEN** 活頁簿有三張工作表，使用者執行 `convert book.xlsx out.csv`
- **THEN** 回錯誤並列出 `Q1, Q2, Q3`，不寫出任何檔案

#### Scenario: One output file for a one-sheet workbook
- **WHEN** 活頁簿只有一張工作表，使用者執行 `convert book.xlsx out.csv`
- **THEN** 寫出 `out.csv`

## ADDED Requirements

### Requirement: List the sheets of a workbook
系統 SHALL 提供 `sheets <file.xlsx>` 命令，依活頁簿順序每行印出一個工作表名稱。

#### Scenario: Listing
- **WHEN** 活頁簿有 `Q1`、`Q2`、`Q3`，使用者執行 `sheets book.xlsx`
- **THEN** 輸出三行 `Q1`、`Q2`、`Q3`
