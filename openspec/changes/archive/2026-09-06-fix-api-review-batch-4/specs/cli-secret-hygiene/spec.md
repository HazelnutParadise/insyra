## ADDED Requirements

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
