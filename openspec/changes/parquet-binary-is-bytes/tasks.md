# Tasks: parquet-binary-is-bytes

## 1. 先重現
- [x] 1.1 測試：同內容的 `Binary` 欄與 `String` 欄讀回來型別與值完全相同，分不出來。
- [x] 1.2 測試：同一個 `Binary` 欄，一列是合法 UTF-8、一列不是，顯示方式不同。
- [x] 1.3 測試：讀進來再寫出去，`Binary` 欄變成字串欄。

## 2. 實作
- [x] 2.1 `getVal` 的三個二進位分支回傳 `[]byte`。
- [x] 2.2 `Read` 與 `recordToDataTable` 對 `[]any` 改用 `NewDataList().Append(vals...)`，避免把 `[]byte` 攤平；其他型別維持原路徑。
- [x] 2.3 `inferArrowType` 認得 `[]byte`，`appendValue` 補 `*array.BinaryBuilder`。

## 3. 測試
- [x] 3.1 兩欄型別分得開，位元組完整。
- [x] 3.2 整欄顯示一致（都是十六進位）。
- [x] 3.3 round trip 後仍是 Arrow 二進位型別，位元組相同。
- [x] 3.4 既有的外來型別測試仍然綠。

## 4. 收尾
- [x] 4.1 `gofmt -l .`、`go build ./...`、`go test ./...`、`golangci-lint run`。
- [x] 4.2 兩份 CHANGELOG。
- [x] 4.3 `Docs/parquet.md` 型別表與二進位說明；`skills/insyra/SKILL.md`。
- [x] 4.4 `AGENTS.md` 移除「讀回來再寫出去仍會換型別」中已由本次解決的部分。
- [x] 4.5 `delivery-status.md`。
