# Tasks: py-wide-integers-refused

## 1. 測試先行

- [x] 1.1 `py/wide_integer_refusal_test.go`：超過 64 位的整數在 `any`、slice、map、`any` 欄位、整數欄位、表格與清單儲存格報錯；在 float 欄位取近似值；`10**400` 的訊息；64 位以內的整數與浮點數不變。在舊程式上確認失敗
- [x] 1.2 `py/environment_setup_test.go`：真實環境加上 `2**64 + 4` 讀進 `any`、`float64` 與 DataFrame 的情況

## 2. 實作

- [x] 2.1 `py/pyresult.go`：`restoreNumbers` 把這種整數留成 `wideInteger`；`numberTooLarge` 對整數改用新的說法
- [x] 2.2 `py/pyresult_decode.go`：float 欄位取近似值，其餘每個落點報錯
- [x] 2.3 以真實環境跑 1.2

## 3. 文件與紀錄

- [x] 3.1 `Docs/py.md`：整數規則
- [x] 3.2 `CHANGELOG.md`、`CHANGELOG_TW.md`：`### py` 段落末尾新增條目

## 4. 驗證

- [x] 4.1 `gofmt -s -l`、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`、`openspec validate py-wide-integers-refused --strict`
