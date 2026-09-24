# Tasks: one-value-one-cell

## 1. 先重現
- [x] 1.1 測試：`NewDataList` 無法把一個切片放進單一格子，而 `Append` 可以，兩者不一致。

## 2. 實作
- [x] 2.1 `Cell(v any) any` 與未匯出的 marker、`unwrapCell`。
- [x] 2.2 `flattenWithNilSupport` 認得 marker，取出內容且不攤平。
- [x] 2.3 六個寫入入口拆 marker：`Append`、`Update`、`InsertAt`、`ReplaceFirst`、`ReplaceAll`、`UpdateElement`，以及兩個以 map 傳值的列附加。
- [x] 2.4 搜尋端拆 marker（`valueMatcher`／`equalCell`），讓 `Count(Cell(x))` 與 `Count(x)` 一致。

## 3. 測試
- [x] 3.1 `NewDataList(Cell([]int{1,2}), 3, "a")` 得到 3 格且第一格型別未被包裝。
- [x] 3.2 未標記的切片仍然攤平。
- [x] 3.3 每個寫入入口用標記與不用標記結果相同，且格子裡不是 marker。
- [x] 3.4 `Count(Cell(x))` 與 `Count(x)` 相同。
- [x] 3.5 marker 不會出現在 `Show`／`ToJSON` 的輸出裡。

## 4. 收尾
- [x] 4.1 `gofmt -l .`、`go build ./...`、`go test ./...`、`golangci-lint run`。
- [x] 4.2 兩份 CHANGELOG。
- [x] 4.3 `Docs/DataList.md`、`Docs/DataTable.md` 說明攤平規則與 `Cell`；skill 一併更新。
- [x] 4.4 `AGENTS.md` 移除「只有建構子把 `[]byte` 當數字清單」那條 follow-up 中已由 `Cell` 回答的部分。
- [x] 4.5 `delivery-status.md`。
