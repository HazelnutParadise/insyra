## ADDED Requirements

### Requirement: LayerNorm takes a typed size

`LayerNorm(dim int)` SHALL build a layer normalizing the last dimension, of size `dim`, and `LayerNormShape(dims []int)` SHALL build one normalizing the trailing `len(dims)` dimensions, copying `dims`. A non-positive size or an empty shape SHALL be reported by `Build`, and so by `NewSequential` naming the layer. `NewLayerNorm(dims interface{})` SHALL remain for one release as a Deprecated constructor naming both, building `LayerNorm` for an `int`, `LayerNormShape` for a `[]int`, and for any other value the layer it built before this change.

#### Scenario: A multi-dimensional normalized shape
- **WHEN** 以 `LayerNormShape([]int{2, 3})` 建立層，之後修改傳入的 slice，再對 `[N, 2, 3]` 的輸入做 forward
- **THEN** 權重與偏差的形狀是 `[2 3]`，結果與改動前 `LayerNorm([]int{2, 3})` 相同，修改 slice 不影響層

#### Scenario: A size that cannot be normalized
- **WHEN** 以 `LayerNorm(0)` 或 `LayerNormShape(nil)` 建立 `Sequential`
- **THEN** `NewSequential` 回傳指出該層的錯誤

#### Scenario: The deprecated constructor keeps its meaning
- **WHEN** 以 `4`、`[]int{2, 3}` 與 `int64(4)` 呼叫 `NewLayerNorm`
- **THEN** 前兩者分別與 `LayerNorm(4)`、`LayerNormShape([]int{2, 3})` 相同，第三者與改動前一樣在 `Build` 時回傳錯誤
