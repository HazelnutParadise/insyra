# Tasks: timestamps-are-times

## 1. Tests first

- [x] 1.1 新測試 `timestamps_test.go`：注入時鐘後，list 與 table 的 `CreationTime`／`LastModifiedTime` 在同一秒內的兩次修改可分辨；時鐘倒退時修改時間不倒退；舊 getter 回傳 Unix 秒；`IsTheSameAs` 對晚兩秒做的 clone 為 true、名稱不同為 false、nil 為 false（舊程式碼沒有新方法，編譯失敗即為紅。舊的 `IsTheSameAs` 另以一次性程式在 1e9b11b6 實測：clone 在同一秒內為 true，跨過秒界為 false；原本的 `TestDataListIsTheSameAs` 也只靠修改時間 +1 判定不同）
- [x] 1.2 `TestDataListGetCreationTimestamp` 改用注入的時鐘，拿掉 `time.Sleep(1 * time.Second)`

## 2. Implementation

- [x] 2.1 新增未匯出的 `clock`，所有寫入時間戳的地方改存 `clock().UnixNano()`
- [x] 2.2 `DataList`／`DataTable` 新增 `CreationTime`、`LastModifiedTime`；舊 getter 標 Deprecated，回傳 Unix 秒
- [x] 2.3 `DataList.IsTheSameAs` 不再比時間
- [x] 2.4 `interfaces.go`：`IDataList`、`IDataTable` 加入兩個新方法
- [x] 2.5 `ml/models_test.go` 的 `bitwiseEqual` 會逐欄比對結果裡 DataTable 的未匯出欄位，連時間戳也比。原本兩個結果在同一秒內算出來，所以剛好相等，跨秒界時會失敗；改成奈秒後每次都會失敗。改成跳過這兩個記錄時間的欄位

## 3. Docs, changelog, ledger

- [x] 3.1 `Docs/DataList.md`、`Docs/DataTable.md`：結構說明、新方法、舊方法標 Deprecated、`IsTheSameAs` 的語意
- [x] 3.2 `CHANGELOG.md`／`CHANGELOG_TW.md`：Core，BREAKING
- [x] 3.3 `AGENTS.md` follow-up：下一版移除兩個舊 getter。`Docs/DataTable.md` 說 `Clone` 保留建立時間，但程式碼和 `DataList.Clone` 一樣給新的時間，所以改的是文件
- [x] 3.4 `api-review.md`：D-12、TS-15 標為已修正；`delivery-status.md`

## 4. Verification

- [x] 4.1 gofmt、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`、`openspec validate timestamps-are-times --strict`
