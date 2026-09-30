# Tasks: nn-custom-loss

## 1. Tests first

- [x] 1.1 新測試：呼叫 `tape.MSELoss` 的 `CustomLoss` 與 `nn.MSE{}` 訓練與 validation 損失逐位元相同；用 `tape.Mul` 組出的損失會訓練；`Validate` 在訓練與 validation 都被呼叫，錯誤帶 loss 名稱；缺少 `Loss`、回傳 nil、非純量、int64、tape 外算出的純量都回傳錯誤且不 panic；空白 `Name` 顯示 `CustomLoss`（舊程式碼沒有 `CustomLoss`，即為紅）

## 2. Implementation

- [x] 2.1 `nn/fit.go`：`CustomLoss` 實作 `LossSpec`；`validateFitConfig` 在任何 batch 前拒絕缺少 `Loss` 的設定；`CustomLoss` 檢查 `Loss` 的回傳值；`LossSpec` 的說明提到 `CustomLoss`

## 3. Docs, changelog, ledger

- [x] 3.1 `Docs/nn.md`：loss 選擇器段落加上 `CustomLoss`、欄位、會被拒絕的情況與範例
- [x] 3.2 `CHANGELOG.md`／`CHANGELOG_TW.md`：`ml` 與 `nn`
- [x] 3.3 `api-review.md`：NN-2 的 loss 部分；`delivery-status.md`

## 4. Verification

- [ ] 4.1 gofmt、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`、`openspec validate nn-custom-loss --strict`
