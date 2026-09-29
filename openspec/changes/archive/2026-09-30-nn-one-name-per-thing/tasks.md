# Tasks: nn-one-name-per-thing

## 1. Tests first

- [x] 1.1 新測試：解析 `nn` 原始碼，每個 Deprecated 名稱的 doc comment 都有 `Deprecated: use <替代名稱>` 與移除說明（舊程式碼沒有，即為紅）
- [x] 1.2 新測試：十二個 `New…` 與對應的原名在同一種子下建出參數相同的層；三個 loss 別名訓練結果相同；別名型別可互換、常數值相同；`Data` 對 int64 張量仍回傳 nil 而 `Float32Data` 回傳錯誤；`NewFloat32Tensor`、`NewTensorWithDType(DTypeFloat32, …)` 與 `NewTensor` 結果相同，`NewTensorWithDType(DTypeInt64, …)` 回傳錯誤

## 2. Implementation

- [x] 2.1 `nn/layers.go`、`nn/layers_catalog.go`、`nn/layers_attention.go`：十二個 `New…` 標 Deprecated
- [x] 2.2 `nn/fit.go`、`nn/protocol.go`、`nn/kernels.go`、`nn/tensor.go`：別名、常數、`NewFloat32Tensor`、`NewTensorWithDType`、`Tensor.Data` 標 Deprecated
- [x] 2.3 呼叫端：`nn` 的程式與測試、`accel/internal/wgpu` 的測試改用保留的名稱，只有檢查 Deprecated 名稱的測試仍使用舊名

## 3. Docs, changelog, ledger

- [x] 3.1 `Docs/nn.md`：loss 名稱只寫一種、張量讀取改用 `Float32Data`、說明哪些名稱已 Deprecated
- [x] 3.2 `CHANGELOG.md`／`CHANGELOG_TW.md`：`ml` 與 `nn`
- [x] 3.3 `AGENTS.md`：下一版移除 Deprecated 名稱的 follow-up
- [x] 3.4 `api-review.md`：NN-1 的名稱部分；`delivery-status.md`

## 4. Verification

- [x] 4.1 gofmt、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`、`openspec validate nn-one-name-per-thing --strict`
