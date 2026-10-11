# Tasks: nn-layernorm-typed-constructors

## 1. Tests first

- [x] 1.1 新測試：`LayerNormShape([]int{2, 3})` 的參數形狀與 forward 結果，以及建立後修改 slice 不影響層；`LayerNorm(0)`、`LayerNorm(-2)`、`LayerNormShape(nil)`、空 shape 與含 0 的 shape 在 `NewSequential` 回傳錯誤；`NewLayerNorm` 的 int、[]int 與字串輸入維持原行為（舊程式碼沒有 `LayerNormShape`，即為紅）

## 2. Implementation

- [x] 2.1 `nn/layers_catalog.go`：新增 `LayerNormShape(dims []int)`；`LayerNorm` 保留 `interface{}`，`[]int` 轉交 `LayerNormShape`；`NewLayerNorm` 標為 Deprecated，行為與 `LayerNorm` 相同

## 3. Docs and changelog

- [x] 3.1 `Docs/nn.md`：層目錄表格的 `LayerNorm` 與 `LayerNormShape`
- [x] 3.2 `CHANGELOG.md`／`CHANGELOG_TW.md`：`nn`
