# sql-identifier-quoting Specification

## Purpose
每一個 SQL 語句都為識別字加引號。

## Requirements
### Requirement: Every statement quotes its identifiers

`ToSQL` 在 SQLite 上查詢既有欄位時 SHALL 對資料表名稱使用 `quoteSQLIdent`，與檔案中其他語句一致。含空白或引號的資料表名稱 SHALL 能被建立並附加。

#### Scenario: Table name with a space
- **WHEN** 先以 Fail 模式建立名為 `my table` 的資料表，再以 Append 模式寫入
- **THEN** 兩次都成功，資料表共有兩次寫入的列數

