# Tasks: ml-fit-returns-concrete-types

## 1. Tests first

- [x] 1.1 新測試：包裝 `stats` 的十四個 `Fit*` 的結果各自指派給具體型別的變數，直接讀 `Result`；再把回傳值放進改動前的介面型別並呼叫（舊程式碼無法編譯，即為紅）。樹與整體模型的六個函式本來就回傳具體型別
- [x] 1.2 新測試：對二十種回傳型別的 nil 指標呼叫每個匯出方法，要求不 panic 且回傳錯誤或空值；`ml.ExportONNX(w, (*ml.LinearModel)(nil))` 回傳錯誤且不寫入（舊程式碼 panic，即為紅）

## 2. Implementation

- [x] 2.1 `ml/models.go`：十四個 `Fit*` 改回傳具體型別；每個型別加 nil 安全的 `Features`，`Predict`／`PredictProba`／`Transform` 在 nil 時回傳錯誤；以 `var _` 在編譯期斷言每個型別實作的介面
- [x] 2.2 `ml/decision_tree.go`、`ml/random_forest.go`、`ml/gradient_boosting.go`：樹與整體模型的 `Features` 與其餘方法在 nil 時不 panic
- [x] 2.3 `ml/onnx_export.go`：`ExportONNX` 在分派前以 `isNilPointer` 拒絕 nil 指標
- [x] 2.4 呼叫端：`ml` 與 `nn` 測試裡的 `Fit: ml.FitLinearRegression` 改成 closure；`ml/models_test.go` 對回傳值的型別斷言在具體型別上不合法，一併移除

## 3. Docs, changelog, ledger

- [x] 3.1 `Docs/ml.md`：`Fit*` 回傳具體型別、`Result` 不必斷言、`Estimator` 裡要寫 closure、介面變數裡的 nil 指標
- [x] 3.2 `CHANGELOG.md`／`CHANGELOG_TW.md`：`ml` 與 `nn`，標 BREAKING
- [x] 3.3 `api-review.md`：ML-1 標為已修正；`delivery-status.md`

## 4. Verification

- [ ] 4.1 gofmt、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`、`openspec validate ml-fit-returns-concrete-types --strict`
