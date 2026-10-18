# Tasks: nn-custom-optimizer

## 1. Tests first

- [x] 1.1 `nn/custom_optimizer_test.go`：用 `Update` 寫的 SGD 與 `nn.SGD` 的訓練損失、validation 損失與參數逐位元相同；`Step` 每個 batch 呼叫一次、拿到 `model.Parameters()`、slice 不與 tape 共用；缺少 `Step`、空白 `Name`、`Step` 錯誤；`Update` 寫入、未被 loss 用到時 `grad` 為 nil、nil 參數與 nil 函式（舊程式碼沒有 `CustomOptimizer` 與 `Update`，編譯失敗，即為紅）

## 2. Implementation

- [x] 2.1 `nn/autodiff.go`：`(*Parameter).Update`
- [x] 2.2 `nn/fit.go`：`CustomOptimizer` 實作 `OptimizerSpec`；`OptimizerSpec` 的說明提到 `CustomOptimizer`

## 3. Docs, changelog, ledger

- [x] 3.1 `Docs/nn.md`：optimizer 選擇器段落後加上 `CustomOptimizer` 與 `Parameter.Update`；手寫迴圈段落指向它
- [x] 3.2 `CHANGELOG.md`／`CHANGELOG_TW.md`：`ml` 與 `nn`
- [x] 3.3 `api-review.md`：NN-2 標為已修正；`delivery-status.md`
- [x] 3.4 Agent skills：不需要改，因為原則、流程與文件位置都沒變

## 4. Verification

- [x] 4.1 gofmt、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`、`openspec validate nn-custom-optimizer --strict`
