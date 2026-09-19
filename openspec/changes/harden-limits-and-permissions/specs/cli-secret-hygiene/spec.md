## ADDED Requirements

### Requirement: Bound query parameters do not reach the log

CLI 開啟資料庫連線 SHALL 關閉 gorm 的預設 logger。該 logger 在查詢失敗或過慢時會把綁定參數內插進訊息印出，`WHERE token = ?` 的值因此會出現在終端機與任何收集它的地方。

#### Scenario: A failing query with a secret parameter
- **WHEN** CLI 執行一個帶有敏感參數且會失敗的查詢
- **THEN** 參數值不會出現在輸出中；錯誤由 CLI 自己回報
