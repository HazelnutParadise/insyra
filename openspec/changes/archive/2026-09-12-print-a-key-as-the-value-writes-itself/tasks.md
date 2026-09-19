# Tasks: print-a-key-as-the-value-writes-itself

## 1. 先重現
- [x] 1.1 測試：實作 `fmt.Stringer` 的格子值，其替身顯示成內部欄位而不是它自己的文字。

## 2. 實作
- [x] 2.1 `UncomparableKey` 多一個未匯出的顯示欄位；`ToMapKey` 在值實作 `fmt.Stringer` 時填入。
- [x] 2.2 `String()` 優先用它，套用同一個截斷上限，沒有就退回編碼內容。
- [x] 2.3 文件寫明它參與 `==` 為何安全，以及不具決定性的 `String()` 會怎樣。

## 3. 測試
- [x] 3.1 Stringer 值顯示成自己的文字，不含內部欄位。
- [x] 3.2 沒有 `String()` 的值（`[]byte`、`[]int`、map）顯示不變。
- [x] 3.3 識別不受影響：文字相同但編碼不同的兩個值仍然是兩組。
- [x] 3.4 過長的 `String()` 一樣被截斷。
- [x] 3.5 以實際的 `finance.ScheduleTable` 確認輸出。

## 4. 收尾
- [x] 4.1 `gofmt -l .`、`go build ./...`、`go test ./...`、`golangci-lint run`。
- [x] 4.2 兩份 CHANGELOG。
- [x] 4.3 `AGENTS.md` 移除這條 follow-up；`delivery-status.md`。
