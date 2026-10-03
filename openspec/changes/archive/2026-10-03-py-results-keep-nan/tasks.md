# Tasks: py-results-keep-nan

## 1. Go 端讀取訊息

- [x] 1.1 `py/ipc_result_test.go`：`decodeResultMessage` 讀得出 `NaN`、`Infinity`、`-Infinity`，字串裡的同名文字不變，一般數字與舊解碼相同，放不進 float64 的數字回錯；handler 對解不開的訊息回錯誤確認並把原因存成該次執行的結果。在舊程式上確認失敗
- [x] 1.2 `py/pyresult.go`：`decodeResultMessage`、標記與還原、handler 的錯誤確認；1.1 全部通過

- [x] 1.3 審查後：拒收原因改存 `refusedStore`，只有行程結束又沒收到其他結果時才當錯誤，`waitForResult` 結束時清掉；新測試在前一版上失敗

## 2. 綁定

- [x] 2.1 `py/nonfinite_bind_test.go`：NaN 與無限大綁進 float、`any`、slice、array、map、struct 欄位、表格；int 等放不進的型別回錯；自己解 JSON 的型別回錯不 panic。在舊程式上確認失敗
- [x] 2.2 `py/pyresult_decode.go`：JSON 寫不出非有限數時改走逐部分解碼，float 與 interface 直接設值；2.1 全部通過

## 3. Python 端

- [x] 3.1 `py/main_test.go` 的假 Python 加 `pyjson:` 模式並檢查確認訊息；`py/nonfinite_run_test.go` 走完整 runner；端對端測試加 NaN、無限大與 `10**400`
- [x] 3.2 `py/builtin.go`：`insyra.Return` 在錯誤確認或沒有確認時 raise；以真的環境跑過

## 4. 文件與紀錄

- [x] 4.1 `Docs/py.md`：Return Type Binding 說明 NaN、無限大與讀不了的結果
- [x] 4.2 `CHANGELOG.md`、`CHANGELOG_TW.md`：`### py` 段落末尾新增條目
- [x] 4.3 `AGENTS.md`：刪掉 NaN 結果的 follow-up
- [x] 4.4 `delivery-status.md`：Latest Milestones 最上方新增條目

## 5. 驗證

- [x] 5.1 `gofmt -s -l`、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`（0 issues）、`openspec validate py-results-keep-nan --strict`
