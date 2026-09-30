## ADDED Requirements

### Requirement: Fit trains with a loss the caller computes on the tape

`nn.CustomLoss{Name, Loss, Validate}` SHALL be accepted as `FitConfig.Loss`. `Loss func(tape *Tape, prediction, target *Tensor) (*Tensor, error)` SHALL be required, and `Fit` SHALL refuse a `CustomLoss` without it before any batch runs. `Fit` SHALL call `Validate func(prediction, target *Tensor) error`, when set, before `Loss` on every training batch and on the validation set. `Name` SHALL label the loss in `Fit`'s errors, and an empty `Name` SHALL read `CustomLoss`. A `Loss` result that is nil, not float32, not a scalar, or not produced by an operation on the tape it was given SHALL be an error naming the loss, never a panic. `LossSpec` SHALL keep its unexported methods.

#### Scenario: A custom loss that is a built-in one
- **WHEN** 以同一個種子，分別用 `nn.MSE{}` 與 `Loss` 呼叫 `tape.MSELoss` 的 `nn.CustomLoss` 執行含 validation 的 `Fit`
- **THEN** 每個 epoch 的訓練損失與 validation 損失逐位元相同

#### Scenario: A loss composed from tape operations
- **WHEN** `Loss` 用 `tape.MSELoss` 算出損失，再用 `tape.Mul` 乘上一個純量
- **THEN** 參數會更新，訓練損失下降

#### Scenario: The caller checks its own targets
- **WHEN** `Validate` 對某個 batch 回傳錯誤
- **THEN** `Fit` 回傳包著該錯誤、帶有 loss 名稱的錯誤，且沒有呼叫 `Loss`

#### Scenario: A missing loss function
- **WHEN** 以沒有 `Loss` 的 `nn.CustomLoss{Name: "huber"}` 呼叫 `Fit`
- **THEN** 在任何 batch 之前回傳指出缺少 `Loss` 的錯誤，參數不變

#### Scenario: A result Fit cannot train on
- **WHEN** `Loss` 回傳 nil、形狀為 `[2]` 的張量、int64 張量，或在 tape 之外用 kernel 算出的純量
- **THEN** `Fit` 回傳帶有 loss 名稱的錯誤，不 panic；名稱空白時錯誤裡寫 `CustomLoss`
