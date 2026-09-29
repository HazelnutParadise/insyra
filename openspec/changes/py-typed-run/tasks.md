# Tasks: py-typed-run

## 1. 測試先行

- [x] 1.1 `py/main_test.go`：測試執行檔也能充當 Python，從收到的腳本讀出執行 ID 與 IPC 位址，回傳指定的結果、錯誤、整份腳本，或一直等到被終止
- [x] 1.2 `py/run_test.go`：`Run` 解碼成 struct、`*insyra.DataTable`；Python 報錯時回傳零值與錯誤；佔位字元被替換；期限到時很快回傳 `DeadlineExceeded`；各 `Context` 形式遇到 nil context 回錯不 panic、遇到已取消的 context 立即回傳 `Canceled`；`PipInstallContext` 在 `uv pip` 執行中到期時回傳 `DeadlineExceeded`；`RunCodeWithTimeout` 保留原意且註解寫明 Deprecated；舊的 `RunCode` 仍可用。在舊程式上確認失敗（新函式不存在）
- [x] 1.4 審查後補強：`Run` 在啟動前拒絕 `insyra.DataTable`／`insyra.DataList` 值型別；已送達的結果優先於行程錯誤；已取消的 context 不會啟動環境準備（以刪掉 pyproject 觀察）；runner 的 goroutine 以參數取得直譯器路徑
- [x] 1.3 `waitForResult` 同時看到行程錯誤與行程結束時一律回傳錯誤：100 次都要拿到錯誤，在舊程式上失敗

## 2. 實作

- [x] 2.1 `py/py.go`：`Run[T]`、`PipInstallContext`、`PipUninstallContext`；nil context 與已結束的 context 在開始前處理；plain 形式改呼叫 `Context` 形式，合併兩份執行函式；`RunCodeWithTimeout` 標示 Deprecated；1.2 通過
- [x] 2.2 `py/pyresult.go`：`waitForResult` 在行程結束的分支先取出行程錯誤；1.3 通過

## 3. 文件與紀錄

- [x] 3.1 `Docs/py.md`：`Run[T]` 一節、`PipInstallContext`／`PipUninstallContext`、nil context、`RunCodeWithTimeout` 的 Deprecated 說明與範例改用 `RunCodeContext`
- [x] 3.2 `skills/insyra/SKILL.md`：`Context` 形式的慣例涵蓋安裝 Python 套件
- [x] 3.3 `CHANGELOG.md`、`CHANGELOG_TW.md`：`### py` 段落末尾新增條目
- [x] 3.4 `AGENTS.md`：follow-up，下一個版本移除 `RunCodeWithTimeout`
- [x] 3.5 `api-review.md`：PY-2 標為已修正
- [x] 3.6 `delivery-status.md`：Latest Milestones 最上方新增條目

## 4. 驗證

- [x] 4.1 `gofmt -s -l`、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`（0 issues）、`openspec validate py-typed-run --strict`；在本機以真的 Python 環境（py-pinned-environment 的端對端測試建出的暫存環境）跑一次 `Run[*insyra.DataTable]`
