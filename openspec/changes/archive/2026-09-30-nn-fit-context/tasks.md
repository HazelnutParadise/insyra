# Tasks: nn-fit-context

## 1. Tests first

- [x] 1.1 新測試：在 `Progress` 裡取消後的回傳值、epoch 數與參數（與 `Epochs: 1` 的 `Fit` 逐位元比較）；`Func` 層在第 2 個 batch 取消後不再有第 3 次 forward；已取消的 context 不改參數、不呼叫 `Progress`；過期 deadline；nil context（舊程式碼沒有 `FitContext`，即為紅）

- [x] 1.2 審查補的測試：在某個 epoch 最後一個 batch 取消時，該 epoch 不回報；最後一個 epoch 的 `Progress` 裡取消時回傳全部結果、沒有錯誤；拿掉 batch 迴圈後那次檢查時第一個測試失敗

## 2. Implementation

- [x] 2.1 `nn/fit.go`：`FitContext`，`Fit` 呼叫它；每個 batch 前與 validation 前檢查 context；取消時回傳已完成 epoch 的結果與 `ctx.Err()`

## 3. Docs, changelog, ledger

- [x] 3.1 `Docs/nn.md`：`FitContext`、取消的時點、回傳什麼、模型保留到哪一步，附在 `Progress` 裡停止的例子
- [x] 3.2 `skills/insyra/SKILL.md`：可以取消的工作加上神經網路訓練
- [x] 3.3 `CHANGELOG.md`／`CHANGELOG_TW.md`：`ml` 與 `nn`
- [x] 3.4 `api-review.md`：NN-2 的 context 部分；`delivery-status.md`

## 4. Verification

- [x] 4.1 gofmt、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`、`openspec validate nn-fit-context --strict`
