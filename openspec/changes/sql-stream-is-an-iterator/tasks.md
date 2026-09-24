# Tasks: sql-stream-is-an-iterator

## 1. 測試先紅
- [x] 1.1 既有兩個 `ReadSQLStream` 測試改成 range 寫法。
- [x] 1.2 第一批之後 break、不 cancel：連線池 `InUse` 歸零。
- [x] 1.3 `db` 為 nil：只 yield 一次錯誤。

## 2. 實作
- [x] 2.1 `ReadSQLStream` 改成在呼叫端 goroutine 裡逐批讀取的 `iter.Seq2`，迴圈結束就關閉 rows；移除 `ReadSQLChunk`。

## 3. 文件與紀錄
- [x] 3.1 `Docs/DataTable.md`、skills；兩份 CHANGELOG 標 BREAKING。
- [x] 3.2 全套驗證；`api-review.md` E-9 與檢查清單、`delivery-status.md`；歸檔、寫 Purpose。
