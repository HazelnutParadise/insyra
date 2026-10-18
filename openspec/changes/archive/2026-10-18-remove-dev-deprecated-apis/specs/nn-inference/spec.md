## MODIFIED Requirements

### Requirement: A tensor has one constructor and one accessor per dtype

`NewTensor` SHALL be the float32 tensor constructor, beside `NewInt64Tensor`, `NewStringTensor` and `NewBoolTensor`. `Float32Data`, `Int64Data`, `StringData` and `BoolData` SHALL be the data accessors, each returning an error for a nil tensor or a tensor of another dtype.

#### Scenario: Reading a tensor of another dtype
- **WHEN** 對 int64 張量呼叫 `Float32Data`
- **THEN** 回傳指出 dtype 的錯誤

## REMOVED Requirements

### Requirement: The replaced tensor names stay one release
**Reason**: `NewFloat32Tensor`, `NewTensorWithDType` and `Tensor.Data` were removed in 0.4, the release after the one that deprecated them.
**Migration**: Use `NewTensor` and `Float32Data`.

### Requirement: A dtype, a binding and pooling options have one name each
**Reason**: It named deprecated APIs that were removed in 0.4, and its scenarios cannot be edited in place; it is restated under a new name in this change.
**Migration**: See the requirement "DType, the bound models and PoolOptions are the names".

## ADDED Requirements

### Requirement: DType, the bound models and PoolOptions are the names

`DType`, the `DType…` constants, `BoundClassifier`, `BoundRegressor` and `PoolOptions` SHALL be the names of these things. The earlier names `DataType`, `Float32`, `Float16`, `Float64`, `Classifier`, `Regressor`, `MaxPoolOptions` and `AveragePoolOptions` SHALL NOT exist.

#### Scenario: The dtype constants and the pooling options
- **WHEN** 把 `nn.DTypeFloat32` 用於建立張量，並把 `nn.PoolOptions{}` 傳給 `nn.MaxPool`
- **THEN** 可以編譯並運作
