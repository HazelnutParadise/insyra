# Tasks: nn-layernorm-typed-constructors

## 1. Tests first

- [x] 1.1 新測試：`LayerNorm` 的參數型別是 `int`，`LayerNormShape` 存在；`LayerNormShape([]int{2, 3})` 的參數形狀與 forward 結果，以及建立後修改 slice 不影響層；`LayerNorm(0)` 與 `LayerNormShape(nil)` 在 `NewSequential` 回傳錯誤；`NewLayerNorm` 三種輸入維持原行為（舊程式碼沒有 `LayerNormShape`，即為紅）

## 2. Implementation

- [x] 2.1 `nn/layers_catalog.go`：`LayerNorm(dim int)`、`LayerNormShape(dims []int)`，`NewLayerNorm(dims interface{})` 依型別轉交兩者，doc comment 指名兩者
- [x] 2.2 呼叫端：傳 slice 給 `LayerNorm` 的程式與測試改用 `LayerNormShape`（查過沒有這種呼叫：repo 裡的 `LayerNorm(` 都傳整數常數）

## 3. Docs, changelog, ledger

- [x] 3.1 `Docs/nn.md`：層目錄表格的 `LayerNorm` 與 `LayerNormShape`
- [x] 3.2 `CHANGELOG.md`／`CHANGELOG_TW.md`：`ml` 與 `nn`，標 BREAKING
- [x] 3.3 `api-review.md`：NN-1 的 `LayerNorm` 部分；`delivery-status.md`

## 4. Verification

- [x] 4.1 gofmt、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`、`openspec validate nn-layernorm-typed-constructors --strict`
