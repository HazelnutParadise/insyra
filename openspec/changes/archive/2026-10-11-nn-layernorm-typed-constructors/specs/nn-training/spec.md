## ADDED Requirements

### Requirement: LayerNormShape normalizes trailing dimensions

`LayerNormShape(dims []int)` SHALL build a layer normalizing the trailing `len(dims)` dimensions, copying `dims` so that later changes to the caller's slice do not affect the layer.

#### Scenario: A multi-dimensional normalized shape
- **WHEN** 以 `LayerNormShape([]int{2, 3})` 建立層，之後修改傳入的 slice，再對 `[2, 2, 3]` 的輸入做 forward
- **THEN** 權重與偏差的形狀是 `[2 3]`，結果與 `LayerNormalization` 以同樣參數算出的結果逐位元相同，修改 slice 不影響層

#### Scenario: A size that cannot be normalized
- **WHEN** 以 `LayerNorm(0)`、`LayerNorm(-2)`、`LayerNormShape(nil)`、`LayerNormShape([]int{})` 或 `LayerNormShape([]int{2, 0})` 建立 `Sequential`
- **THEN** `NewSequential` 回傳指出 `layer 0 (LayerNorm)` 的錯誤

### Requirement: LayerNorm and NewLayerNorm keep the inputs they accepted

`LayerNorm` SHALL keep its `interface{}` parameter, building one dimension for an `int` and the same layer as `LayerNormShape` for a `[]int`, and `NewLayerNorm` SHALL keep its old meaning as a Deprecated name for `LayerNorm`.

#### Scenario: The deprecated constructor keeps its meaning
- **WHEN** 以 `4`、`[]int{2, 3}` 與 `"4"` 呼叫 `NewLayerNorm`
- **THEN** 前兩者分別建立與 `LayerNorm(4)`、`LayerNormShape([]int{2, 3})` 相同的權重形狀，第三者在 `Build` 時回傳 dimension 錯誤
