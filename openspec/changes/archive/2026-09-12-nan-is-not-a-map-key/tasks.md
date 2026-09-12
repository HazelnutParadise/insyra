# Tasks: nan-is-not-a-map-key

## 1. 先重現
- [x] 1.1 測試：三個 NaN 的欄位，`Counter` 產生三個取不回來的項目，`Count` 回 3，兩者不一致。
- [x] 1.2 測試：含 NaN 的陣列與結構同樣漏掉。

## 2. 實作
- [x] 2.1 `comparableCell` 改成「可比較且等於自己」，`Comparable()` 先判以免 `==` panic。
- [x] 2.2 `UncomparableKey` 的文件說明涵蓋 NaN。

## 3. 測試
- [x] 3.1 三個 NaN 合併成一項、計數為 3、以替身查得到。
- [x] 3.2 `Count(NaN)` 與 `Counter` 一致。
- [x] 3.3 含 NaN 的陣列與結構同樣處理。
- [x] 3.4 一般值（含不是 NaN 的浮點數、同型別但無 NaN 的結構）仍以自身當 key。

## 4. 收尾
- [x] 4.1 `gofmt -l .`、`go build ./...`、`go test ./...`、`golangci-lint run`。
- [x] 4.2 兩份 CHANGELOG。
- [x] 4.3 `Docs/DataList.md`、`Docs/DataTable.md` 的 Counter 說明補上 NaN。
- [x] 4.4 `delivery-status.md`。
