# Tasks: sql-replace-keeps-old-table

## 1. Tests

- [x] 1.1 `datatable_to_sql_replace_test.go`：在 SQLite 上以 `ToSQL` 做 Replace，成功後資料換成新列，且沒有 `insyra_` 開頭的殘留表（`TestToSQLReplaceKeepsSQLiteTransactionalBehavior`）

## 2. Implementation

- [x] 2.1 `datatable_to_sql.go`：`ToSQLContext` 在 MySQL 的既有表 Replace 走 `replaceThroughStagingTable`，其他情況照舊；`tableReferencedBy` 拒絕被外鍵參照的表（同庫自我參照除外）；`RENAME TABLE` 在交易之外執行，清理語句使用獨立的 context 並以 30 秒為上限；`RENAME` 失敗後在獨立 context 查證是否已換上

## 3. Docs and changelog

- [x] 3.1 `Docs/DataTable.md`：ToSQL 的行為、Replace 與 Returns 說明各資料庫失敗後保留的狀態
- [x] 3.2 `CHANGELOG.md`、`CHANGELOG_TW.md`：`Core` 段落新增條目
