# Tasks: read-write-names

## 1. 實作
- [x] 1.1 測試先紅：`ReadCSVFile`／`ReadCSVString` 預設讀標題列、`NoHeaderRow`、`HasRowNames`、兩包報錯；`StreamCSV` 新參數順序；`ToCSV` 預設寫標題列；舊名字保留舊意思；`ReadJSONFile`、`ToJSONBytes`／`ToJSONString`；`ExcelWriteOptions` 預設寫標題列。
- [x] 1.2 新函式、欄位改名與反向、舊名字改成 Deprecated 包裝；`isr`、CLI 跟著改；既有測試照新名字改寫，舊的「零值等於沒標題列」測試照裁定改成驗證舊名字的意思。

## 2. 文件與紀錄
- [x] 2.1 `Docs/`、`skills/`、兩份 README；兩份 CHANGELOG；`AGENTS.md` follow-up。
- [x] 2.2 全套驗證；`api-review.md` K-14、`delivery-status.md`；歸檔、寫 Purpose；在 #213 回報並關閉。
