# Tasks: py-placeholders-one-pass

## 1. 測試先行

- [x] 1.1 `py/placeholders_test.go`：每種值現在的輸出（字串、bool、`[]int`、`[]float64`、`[]string`、有名稱的 `IDataList`、空 `IDataList`、有欄名列名的 `IDataTable`、map、int、float、nil）；值裡的 `$v2` 不再被替換；`$v10` 依完整編號；超出參數或開頭是 0 的佔位字元照原樣；`%v` 備案改為錯誤；沒用到的參數不轉換。在舊程式上確認新行為的測試失敗、每種值的輸出測試通過

## 2. 實作

- [x] 2.1 `py/py.go`：轉換移到 `pythonLiteral`，`replacePlaceholders` 一次掃描模板、依完整編號替換、只轉換用到的參數；JSON 寫不出的值回傳錯誤；1.1 全部通過

- [x] 2.2 審查後：`[]float64` 裡有 NaN 或無限大時回傳錯誤，跟單獨的 NaN 一樣，不再寫出 Python 不認得的 `NaN`、`+Inf`；新測試在前一版上失敗

## 3. 文件與紀錄

- [x] 3.1 `Docs/py.md`：Run Code With Parameters 說明一次替換、完整編號、超出參數照原樣、無法寫成 Python 值時回錯
- [x] 3.2 `CHANGELOG.md`、`CHANGELOG_TW.md`：`### py` 段落末尾新增安全修正條目
- [x] 3.3 `api-review.md`：PY-2 列註明佔位字元注入已修正
- [x] 3.4 `AGENTS.md`：刪掉「佔位字元替換」follow-up
- [x] 3.5 `delivery-status.md`：Latest Milestones 最上方新增條目

## 4. 驗證

- [x] 4.1 `gofmt -s -l`、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`（0 issues）、`openspec validate py-placeholders-one-pass --strict`
