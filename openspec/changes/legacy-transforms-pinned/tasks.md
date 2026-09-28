# Tasks: legacy-transforms-pinned

## 1. Tests

- [x] 1.1 新測試檔：六組新舊方法在全數值輸入上數值一致，並逐項固定規格列出的差異（長度、nil、NaN、邊界視窗、失敗）。此為特性測試，新舊程式都應通過；若有任何一項與契約不符，照實回報，不改程式也不改期望值

## 2. Docs, ledger

- [x] 2.1 `Docs/DataList.md`：新增六組差異表；六個舊方法各自補上缺值與失敗的行為並連到差異表；`Difference` 失敗時回傳空 list（不是 nil）
- [x] 2.2 `api-review.md`：D-14 註記「經比對六組皆不等價，差異已文件化並以測試固定，是否仍淘汰待擁有者裁定（#222）」
- [x] 2.3 `delivery-status.md`

## 3. Verification

- [x] 3.1 gofmt、`go vet ./...`、`go test ./...`、`golangci-lint run`、`openspec validate legacy-transforms-pinned --strict`
