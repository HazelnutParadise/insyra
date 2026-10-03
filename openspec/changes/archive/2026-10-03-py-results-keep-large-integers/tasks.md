# Tasks: py-results-keep-large-integers

## 1. 解碼訊息

- [x] 1.1 `py/ipc_result_test.go`：超過 2^53 的整數解成 `int64`／`uint64`，2^53 以內與 float 照舊是 float64，超過 64 位元是最接近的 float64；一般訊息照舊走快路徑。在舊程式上確認失敗
- [x] 1.2 `py/pyresult.go`：連續 16 位數字時改走保留數字原文的解碼，`restoreNumbers` 把大整數還原成 `int64`／`uint64`；1.1 全部通過
- [x] 1.3 只把不是小數一部分的長數字算進去（`hasLongInteger`），浮點數訊息不再走慢路徑；19 MB 的小數訊息量測解碼時間

## 2. 綁定

- [x] 2.1 `py/large_integer_test.go`：大整數綁進 `int64`、`uint64`、`any`、`map[string]any`、struct 裡的 `any`、表格；經過 runner 也一樣。在舊程式上確認失敗
- [x] 2.2 `py/pyresult_decode.go`：含有 `any` 的型別逐部分解碼；2.1 全部通過
- [x] 2.3 端對端測試：真的 Python 回傳 `2**53 + 1`、含大整數的 DataFrame，以及照舊是 float64 的小整數；以真的環境跑過

- [x] 2.4 審查後：`any` 裡原本有非 nil 指標時解進指標；整數欄位自己設值並檢查範圍，結果有 `uint64` 或絕對值 2^63 以上的 float64 時不走 JSON；`hasLongInteger` 跳過字串；新測試在前一版上失敗
- [x] 2.5 審查後：`insyra.Return` 逐欄轉換 pandas／polars DataFrame，整數欄旁有小數欄時保住大整數；以真的環境跑過

## 3. 文件與紀錄

- [x] 3.1 `Docs/py.md`：Return Type Binding 說明整數怎麼回來
- [x] 3.2 `CHANGELOG.md`、`CHANGELOG_TW.md`：`### py` 段落末尾新增條目
- [x] 3.3 `AGENTS.md`：follow-up 只留下鍵順序那一半
- [x] 3.4 `delivery-status.md`：Latest Milestones 最上方新增條目

## 4. 驗證

- [x] 4.1 `gofmt -s -l`、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`（0 issues）、`openspec validate py-results-keep-large-integers --strict`
