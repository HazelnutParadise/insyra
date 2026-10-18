## ADDED Requirements

### Requirement: Fit steps with an optimizer the caller writes

`nn.CustomOptimizer{Name, Step}` SHALL be accepted as `FitConfig.Optimizer`, and `Fit` SHALL refuse one without `Step` before any batch runs. `Fit` SHALL call `Step` once per batch, after the reverse pass, with a new slice of every parameter tracked on the model's tape in creation order. A `Step` error SHALL stop `Fit`. `OptimizerSpec` SHALL stay sealed, and the built-in optimizers SHALL keep their results.

#### Scenario: A custom optimizer that is a built-in one
- **WHEN** 以同一個種子，分別用 `nn.SGD{Rate: 0.05}` 與用 `Parameter.Update` 做 `value -= 0.05*grad` 的 `nn.CustomOptimizer` 執行含 validation 的 `Fit`
- **THEN** 每個 epoch 的訓練損失、validation 損失與訓練後的每個參數都逐位元相同

#### Scenario: Step sees every parameter once per batch
- **WHEN** 6 列資料以 `BatchSize: 4` 跑 2 個 epoch
- **THEN** `Step` 被呼叫 4 次，每次拿到的參數依序與 `model.Parameters()` 相同；`Step` 清空它拿到的 slice 不影響下一個 batch

#### Scenario: A missing step function
- **WHEN** 以沒有 `Step` 的 `nn.CustomOptimizer{Name: "lion"}` 呼叫 `Fit`
- **THEN** 在任何 batch 之前回傳指出 `lion` 缺少 `Step` 的錯誤，參數不變；名稱空白時錯誤裡寫 `CustomOptimizer`

#### Scenario: A step that fails
- **WHEN** `Step` 回傳錯誤
- **THEN** `Fit` 停止，回傳包著該錯誤、帶有 optimizer 名稱的錯誤

### Requirement: A parameter can be updated from its gradient

`(*Parameter).Update(fn func(value, grad []float32)) error` SHALL call `fn` with the parameter's own float32 data and its most recent gradient, in the same layout, and the next forward pass SHALL read what `fn` wrote into `value`. `grad` SHALL be nil when the last reverse pass did not reach the parameter or none has run. `Update` SHALL return an error, without calling anything, for a nil parameter or a nil `fn`, and SHALL NOT change the state of the tape's built-in optimizers.

#### Scenario: Writing a parameter from its gradient
- **WHEN** `Backward` 之後對參數呼叫 `Update`，在 `fn` 裡做 `value[i] -= 0.5*grad[i]`
- **THEN** `fn` 拿到的 `grad` 等於 `Tape.Grad` 回傳的值，`Value()` 反映寫入的結果，下一次前向計算讀到新值

#### Scenario: A parameter the loss did not reach
- **WHEN** 對 loss 沒有用到的參數呼叫 `Update`
- **THEN** `fn` 拿到的 `grad` 是 nil

#### Scenario: Nothing to update
- **WHEN** 對 nil 參數呼叫 `Update`，或傳入 nil 函式
- **THEN** 回傳錯誤，不 panic
