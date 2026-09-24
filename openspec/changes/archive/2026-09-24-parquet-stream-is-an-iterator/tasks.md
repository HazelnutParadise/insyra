# Tasks: parquet-stream-is-an-iterator

## 1. 測試先紅
- [x] 1.1 讀完整個檔案：列數正確、沒有錯誤。
- [x] 1.2 第一批之後 break、不 cancel：stream 開的 goroutine 全部結束。
- [x] 1.3 檔案不存在：只 yield 一次 nil 表格與錯誤。
- [x] 1.4 既有的 `TestStreamReportsAnUnreadableColumn` 改成新寫法。

## 2. 實作
- [x] 2.1 原本的雙 channel 本體改名成內部函式，`Stream` 包成 `iter.Seq2`，離開迴圈時取消內部 context。

## 3. 文件與紀錄
- [x] 3.1 `Docs/parquet.md` 兩處、skills；兩份 CHANGELOG 標 BREAKING。
- [x] 3.2 全套驗證；`api-review.md` Q-7、Q-9 與對照列、`delivery-status.md`；歸檔、關 #273。
