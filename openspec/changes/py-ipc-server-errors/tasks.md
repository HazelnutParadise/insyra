# Tasks: py-ipc-server-errors

## 1. 測試先行

- [x] 1.1 `py/ipc_server_test.go`：`TMPDIR` 過長時 `RunCodeContext` 回傳可用 `errors.Is(err, syscall.EINVAL)` 辨識的錯誤，且不準備環境、不啟動 Python；失敗後恢復 `TMPDIR`，下一次取得伺服器成功；兩個同時進行的執行共用同一位址；最後一個執行結束後 socket 檔消失；連線送出結果後會被存進結果表並收到回覆。在舊程式上確認失敗（新函式不存在，編譯失敗）
- [x] 1.2 審查後補強：持有 `pyInitMu` 時同時呼叫 `RunCodeContext` 與 `RunCode`，先準備環境的 runner 會卡在鎖上而被抓到；accept 迴圈因非逾時錯誤停止時要關掉 listener（假 listener 測試，在舊程式上失敗）

## 2. 實作

- [x] 2.1 `py/pyresult.go`：以參照計數的 `acquireIPCServer`／`releaseIPCServer` 取代 `startServer`、`serverOnce`、`serverReady`、`getIPCAddress`；開啟失敗回傳包住 listen 錯誤的 error；移除對全域變數註冊的 `runtime.AddCleanup`
- [x] 2.2 `py/py.go`：兩個執行函式先取得伺服器、結束時釋放，再準備環境；`generateDefaultPyCode` 改為接收位址
- [x] 2.3 `py/init.go`：`pyEnvInit` 不再啟動伺服器；1.1 通過
- [x] 2.4 `py/pyresult.go`：`acceptIPC` 停止前關掉 listener；1.2 通過

## 3. 文件與紀錄

- [x] 3.1 `Docs/py.md`：IPC 伺服器開啟失敗時的回傳，與 socket 檔的存在期間
- [x] 3.2 `CHANGELOG.md`、`CHANGELOG_TW.md`：更正 `### py` 既有條目裡 socket 檔的說法，並在段落末尾新增條目
- [x] 3.3 `api-review.md`：PY-1 註明 IPC 部分已修正，其餘仍待處理
- [x] 3.4 `delivery-status.md`：Latest Milestones 最上方新增條目
- [x] 3.5 檢查 `skills/insyra/SKILL.md`：不談 IPC，本變更不需修改

## 4. 驗證

- [x] 4.1 `gofmt -s -l`、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`（0 issues）、`openspec validate py-ipc-server-errors --strict`
