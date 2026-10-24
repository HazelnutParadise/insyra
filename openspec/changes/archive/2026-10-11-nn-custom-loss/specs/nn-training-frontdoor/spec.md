## ADDED Requirements

### Requirement: Fit trains with a loss the caller computes on the tape

`Fit` SHALL accept `nn.CustomLoss{Name, Loss, Validate}` as `FitConfig.Loss` and train with the loss that `Loss` computes on the tape it is given.

#### Scenario: A custom loss that is a built-in one
- **WHEN** 以同一個種子，分別用 `nn.MSE{}` 與 `Loss` 呼叫 `tape.MSELoss` 的 `nn.CustomLoss` 執行含 validation 的 `Fit`
- **THEN** 每個 epoch 的訓練損失與 validation 損失逐位元相同

#### Scenario: A loss composed from tape operations
- **WHEN** `Loss` 用 `tape.MSELoss` 算出損失，再用 `tape.Mul` 乘上一個純量
- **THEN** 參數會更新，訓練損失下降

### Requirement: Fit checks a custom loss before and while it trains

`Fit` SHALL refuse a `CustomLoss` without `Loss` before any batch runs, and SHALL return an error naming the loss, never panic, when `Loss` returns a result it cannot train on.

#### Scenario: A missing loss function
- **WHEN** 以沒有 `Loss` 的 `nn.CustomLoss{Name: "huber"}` 呼叫 `Fit`
- **THEN** 在任何 batch 之前回傳指出缺少 `Loss` 的錯誤，參數不變

#### Scenario: A result Fit cannot train on
- **WHEN** `Loss` 回傳 nil、形狀為 `[2]` 的張量、int64 張量，或在 tape 之外用 kernel 算出的純量
- **THEN** `Fit` 回傳帶有 loss 名稱的錯誤，不 panic

#### Scenario: An empty name
- **WHEN** `Name` 為空白的 `CustomLoss` 回傳上述任一錯誤
- **THEN** 錯誤裡的名稱是 `CustomLoss`

### Requirement: A custom loss's Validate runs before Loss

`Fit` SHALL call `Validate`, when set, before `Loss` on every training batch and on the validation set, and SHALL wrap its error with the loss's name.

#### Scenario: The caller checks its own targets
- **WHEN** `Validate` 對某個 batch 回傳錯誤
- **THEN** `Fit` 回傳包著該錯誤、帶有 loss 名稱的錯誤，且沒有呼叫 `Loss`
