## ADDED Requirements

### Requirement: A layer and a loss have one name each

The layer constructors SHALL be `Dense`, `ReLU`, `NewSigmoid`, `NewTanh`, `NewGelu`, `Dropout`, `NewFlatten`, `Func`, `Conv2D`, `MaxPool2D`, `AvgPool2D`, `GlobalAvgPool`, `BatchNorm2D`, `LayerNorm`, `Embedding`, `MultiHeadAttention` and `Residual`; the `New` prefix SHALL appear only where the bare name is a kernel function. `NewDense`, `NewReLU`, `NewDropout`, `NewFunc`, `NewMultiHeadAttention`, `NewConv2D`, `NewMaxPool2D`, `NewAvgPool2D`, `NewGlobalAvgPool`, `NewBatchNorm2D`, `NewLayerNorm` and `NewEmbedding` SHALL remain for one release as Deprecated constructors that build the same layer as their bare name. The `FitConfig.Loss` selectors SHALL be `CrossEntropy`, `MSE` and `BCEWithLogits`; `SoftmaxCrossEntropy`, `MSELoss` and `BCEWithLogitsLoss` SHALL remain for one release as Deprecated aliases of them. Each Deprecated doc comment SHALL name its replacement.

#### Scenario: A deprecated twin builds the same layer
- **WHEN** 以同一個種子分別用 `NewDense(3, 4)` 與 `Dense(3, 4)` 建立 `Sequential`
- **THEN** 參數的形狀、順序與數值相同；`NewDense` 的 doc comment 有指名 `Dense` 的 `Deprecated:` 段落

#### Scenario: A deprecated loss alias trains the same way
- **WHEN** 以同一個種子分別用 `nn.MSELoss{}` 與 `nn.MSE{}` 執行 `Fit`
- **THEN** 每個 epoch 的訓練損失相同
