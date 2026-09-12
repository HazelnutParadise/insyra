# Tasks: a-decimal-is-a-number

## 1. 先重現
- [x] 1.1 測試：十進位欄的 `Mean` 回 `NaN` 且 `Err()` 是 nil。
- [x] 1.2 測試：`IsNumeric` 與數值讀取都拒絕十進位。
- [x] 1.3 測試：混合十進位與浮點數的欄位排序分成兩段。

## 2. 實作
- [x] 2.1 `internal/utils` 以形狀（能報文字與小數位數、且文字可解析為數字）辨認十進位，`ToFloat64`／`ToFloat64Safe` 讀得到。
- [x] 2.2 根套件的 `IsNumeric` 跟著一致。
- [x] 2.3 `internal/algorithms` 的排序序位改到數值那一組；兩個十進位之間仍走 `decimal.Cmp`。

## 3. 測試
- [x] 3.1 `finance.ScheduleTable` 的欄位 `Mean`／`Sum` 得到數值。
- [x] 3.2 `IsNumeric` 與讀取一致。
- [x] 3.3 混合欄位依值交錯排序。
- [x] 3.4 兩個十進位之間超出 `float64` 精度的差異仍分辨得出。
- [x] 3.5 只有相同方法但文字不是數字的型別不被當成數值。
- [x] 3.6 身分與計數不變：十進位仍然不能當 map key。

## 4. 收尾
- [x] 4.1 `gofmt -l .`、`go build ./...`、`go test ./...`、`golangci-lint run`。
- [x] 4.2 兩份 CHANGELOG。
- [x] 4.3 `Docs/finance.md` 與 `Docs/parquet.md` 說明十進位欄可做數值運算及精度界線。
- [x] 4.4 `AGENTS.md` 移除該 follow-up；在 #247 留言；`delivery-status.md`。
