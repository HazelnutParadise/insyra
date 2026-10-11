# Tasks: parquet-damaged-row-groups

## 1. parquet

- [x] 1.1 `readTableFrom`、`streamAsArrowRecord` 與 `streamArrowRecordsFrom` 比對讀到的列數與 metadata 的列數，不同時回傳指名檔案、兩個列數與未完整讀出之 row group 的錯誤（`parquet/damaged_rows_test.go`）
- [x] 1.2 `ApplyCCL` 遇到損壞檔案時回傳錯誤、原檔位元組不變、不留暫存檔（`parquet/damaged_rows_test.go`）

## 2. Changelog

- [x] 2.1 `CHANGELOG.md`／`CHANGELOG_TW.md`：`parquet`
