## ADDED Requirements

### Requirement: A tensor has one constructor and one accessor per dtype

`NewTensor` SHALL be the float32 tensor constructor and `Float32Data` the float32 accessor, while `NewFloat32Tensor`, `NewTensorWithDType` and `Tensor.Data` SHALL remain for one release as Deprecated names, each naming its replacement.

#### Scenario: Reading a tensor of another dtype
- **WHEN** 對 int64 張量呼叫 `Float32Data`
- **THEN** 回傳指出 dtype 的錯誤；已 Deprecated 的 `Data` 仍回傳 nil

#### Scenario: The deprecated constructors keep their meaning
- **WHEN** 以同樣的形狀與資料呼叫 `NewFloat32Tensor`、`NewTensorWithDType(DTypeFloat32, …)` 與 `NewTensor`，再以 `DTypeInt64` 呼叫 `NewTensorWithDType`
- **THEN** 前三者得到相同的張量，最後一個回傳錯誤；兩個 Deprecated 函式的 doc comment 都指名 `NewTensor`

### Requirement: A dtype, a binding and pooling options have one name each

`DType`, the `DType…` constants, `BoundClassifier`, `BoundRegressor` and `PoolOptions` SHALL be the names of these things, while `DataType`, `Float32`, `Float16`, `Float64`, `Classifier`, `Regressor`, `MaxPoolOptions` and `AveragePoolOptions` SHALL remain for one release as Deprecated aliases or constants.

#### Scenario: The deprecated aliases are the same types and values
- **WHEN** 比較 `nn.Float32` 與 `nn.DTypeFloat32`，並把 `nn.MaxPoolOptions{}` 傳給 `nn.MaxPool`
- **THEN** 值相同、可以編譯；每個 Deprecated 名稱的 doc comment 都指名替代名稱
