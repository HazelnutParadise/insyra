# Tasks: save-writes-excel-sheets

## 1. 函式庫
- [x] 1.1 測試先紅：新檔、加表不動其他表、預設拒絕且檔案位元組不變、replace 保留位置與其他表、預設名稱 `Sheet1`、`WriteExcel` 寫到 buffer。
- [x] 1.2 `ToExcel`、`WriteExcel`、`ExcelWriteOptions`、`SheetExistsPolicy`、`ErrSheetExists`；透過暫存檔寫入。

## 2. CLI
- [x] 2.1 測試先紅：存 `.xlsx`、存兩次第二次拒絕並提示、`if-exists replace` 成功、`.xls` 被拒、`sheet` 用在 CSV 被拒。
- [x] 2.2 `save` 加 Excel 分支與兩個選項，Usage 與說明同步。
- [x] 2.3 沒指定工作表而 `Sheet1` 已存在時，不自動改成 `Sheet2`，拒絕訊息同時提示 `sheet <name>` 與 `if-exists replace`（擁有者 2026-09-25 裁定）；測試先紅。

## 3. 文件與紀錄
- [x] 3.1 `Docs/DataTable.md`、`Docs/cli-dsl.md`、兩個 skills；兩份 CHANGELOG。
- [x] 3.2 全套驗證；`api-review.md` CLI-21、`delivery-status.md`；歸檔、寫 Purpose；關 #329。
