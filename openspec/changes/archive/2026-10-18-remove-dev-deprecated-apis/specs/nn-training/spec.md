## REMOVED Requirements

### Requirement: The replaced layer constructors stay one release
**Reason**: `NewDense`, `NewReLU`, `NewDropout`, `NewFunc`, `NewMultiHeadAttention`, `NewConv2D`, `NewMaxPool2D`, `NewAvgPool2D`, `NewGlobalAvgPool`, `NewBatchNorm2D`, `NewLayerNorm` and `NewEmbedding` were removed in 0.4, the release after the one that deprecated them.
**Migration**: Use the bare name: `Dense`, `ReLU`, `Dropout`, `Func`, `MultiHeadAttention`, `Conv2D`, `MaxPool2D`, `AvgPool2D`, `GlobalAvgPool`, `BatchNorm2D`, `LayerNorm`, `Embedding`.

### Requirement: NewLayerNorm stays one release
**Reason**: `NewLayerNorm` was removed in 0.4, the release after the one that deprecated it.
**Migration**: Use `LayerNorm(dim)` for an `int` and `LayerNormShape(dims)` for a `[]int`.

### Requirement: A layer and a loss have one name each
**Reason**: It named deprecated APIs that were removed in 0.4, and its scenarios cannot be edited in place; it is restated under a new name in this change.
**Migration**: See the requirement "Layer constructors take the bare name".

### Requirement: The losses have one name each
**Reason**: It named deprecated APIs that were removed in 0.4, and its scenarios cannot be edited in place; it is restated under a new name in this change.
**Migration**: See the requirement "Fit loss selectors are named once".

## ADDED Requirements

### Requirement: Layer constructors take the bare name

The layer constructors SHALL be `Dense`, `ReLU`, `NewSigmoid`, `NewTanh`, `NewGelu`, `Dropout`, `NewFlatten`, `Func`, `Conv2D`, `MaxPool2D`, `AvgPool2D`, `GlobalAvgPool`, `BatchNorm2D`, `LayerNorm`, `Embedding`, `MultiHeadAttention` and `Residual`; the `New` prefix SHALL appear only where the bare name is a kernel function.

#### Scenario: The bare constructor builds the layer
- **WHEN** 以 `Dense(3, 4)` 建立 `Sequential`
- **THEN** 它有一個形狀為 3×4 的權重與一個長度 4 的偏差，且沒有 `NewDense`

### Requirement: Fit loss selectors are named once

The `FitConfig.Loss` selectors SHALL be `CrossEntropy`, `MSE` and `BCEWithLogits`. The earlier selectors `SoftmaxCrossEntropy`, `MSELoss` and `BCEWithLogitsLoss` SHALL NOT exist at package level; `Tape.SoftmaxCrossEntropy`, `Tape.MSELoss` and `Tape.BCEWithLogitsLoss` are the tape's loss methods and SHALL remain.

#### Scenario: A selector trains
- **WHEN** 以 `nn.MSE{}` 作為 `FitConfig.Loss` 執行 `Fit`
- **THEN** 每個 epoch 回報訓練損失，這個損失與 `Tape.MSELoss` 對同一批資料算出的相同
