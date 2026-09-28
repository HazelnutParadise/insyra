# cli-secret-hygiene Specification

## Purpose
資料庫密碼不得以明文進入 history 或匯出檔：`db connect` 在寫入任何歷史紀錄之前遮罩密碼，history 檔只有使用者本人可讀。

## Requirements

### Requirement: Database passwords never persist in clear text

`db connect` 這一行寫入 history（one-shot、REPL、DSL session）前 SHALL 經 `SanitizeHistoryLine` 遮罩 URL、`user:pass@`、`password=`／`pwd=` 三種形式的密碼；`=` 前後可有空格；值加了引號或含空格時 SHALL 整段遮罩，直到結尾引號、`;` 或下一個 `key=` 為止，單引號內以反斜線跳脫的字元與大括號內的 `}}` SHALL 視為值的一部分；history 檔 SHALL 以 0600 建立。

#### Scenario: URL DSN
- **WHEN** 執行 `db connect a mysql://alice:S3cretPW@host/db`
- **THEN** history.txt 含 `db connect a` 但不含 `S3cretPW`

#### Scenario: A password containing spaces
- **WHEN** 執行 `db connect pg "host=h password=a b"`
- **THEN** history.txt 記錄為 `db connect pg "host=h password=***"`，不含 `a b`，也不殘留 ` b`

#### Scenario: An escaped quote or spaces around the equals sign
- **WHEN** 執行 `db connect pg "host=h password='it\'s s3cr3tB' port=5"` 或 `db connect pg "host=h password = s3cr3tA port=5432"`
- **THEN** history.txt 分別記錄為 `db connect pg "host=h password=*** port=5"` 與 `db connect pg "host=h password = *** port=5432"`，不含密碼的任何部分

### Requirement: Bound query parameters do not reach the log

CLI 開啟資料庫連線 SHALL 關閉 gorm 的預設 logger。該 logger 在查詢失敗或過慢時會把綁定參數內插進訊息印出，`WHERE token = ?` 的值因此會出現在終端機與任何收集它的地方。

#### Scenario: A failing query with a secret parameter
- **WHEN** CLI 執行一個帶有敏感參數且會失敗的查詢
- **THEN** 參數值不會出現在輸出中；錯誤由 CLI 自己回報
