## ADDED Requirements

### Requirement: A layer and a loss have one name each

The twelve `New…` layer twins and the `FitConfig.Loss` aliases SHALL remain for one release as Deprecated names, each building the same layer or loss as its replacement, and each doc comment SHALL name that replacement.

#### Scenario: A deprecated twin builds the same layer
- **WHEN** 以同一個種子分別用 `NewDense(3, 4)` 與 `Dense(3, 4)` 建立 `Sequential`
- **THEN** 參數的形狀、順序與數值相同；`NewDense` 的 doc comment 指名 `Dense`

#### Scenario: A deprecated loss alias trains the same way
- **WHEN** 以同一個種子分別用 `nn.MSELoss{}` 與 `nn.MSE{}` 執行 `Fit`
- **THEN** 每個 epoch 的訓練損失相同
