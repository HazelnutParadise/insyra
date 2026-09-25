# Tasks: exported-functions-are-functions

## 1. 實作
- [x] 1.1 測試先紅：掃描全模組，匯出變數不得存函式字面值或本模組宣告的函式（現況應列出 `ToFloat64`、`ToFloat64Safe`、`ReadSlice2D`、`mkt.CAI`）；`Slice2DToDataTable` 與 `ReadSlice2D` 結果相同。
- [x] 1.2 四個名字改成 `func`；`ReadSlice2D` 承接實作，`Slice2DToDataTable` 標 Deprecated；`isr`、`py` 改呼叫 `ReadSlice2D`。

## 2. 文件與紀錄
- [x] 2.1 `Docs/DataTable.md`、`Docs/mkt.md`、skills；兩份 CHANGELOG（Core、`mkt`）。
- [x] 2.2 `AGENTS.md` 新增下一版移除 `Slice2DToDataTable` 的 follow-up。
- [x] 2.3 全套驗證；`api-review.md` K-12、`delivery-status.md`；歸檔、寫 Purpose；關 #211。
