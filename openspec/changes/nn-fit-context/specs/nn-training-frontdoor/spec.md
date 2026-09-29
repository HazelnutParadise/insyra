## ADDED Requirements

### Requirement: Training can be cancelled between batches

`(*Sequential).FitContext(ctx, x, y, cfg)` SHALL train exactly as `Fit` does, and `Fit` SHALL be `FitContext` with `context.Background()`. `FitContext` SHALL check `ctx` before every batch and before each epoch's validation. When `ctx` is done, it SHALL return `ctx.Err()` unwrapped and a `FitResult` listing only the epochs whose batches, validation and `Progress` call completed, with `Elapsed` set; the parameters SHALL keep every optimizer step taken before the check. A run whose every epoch finished SHALL return its result and no error, even if `ctx` was cancelled during the last `Progress` call. A context already done at the call SHALL leave the model and its tape unchanged. A nil context SHALL be refused with an error before anything changes.

#### Scenario: Stopping after an epoch
- **WHEN** 以同一個種子執行 3 個 epoch 的 `FitContext`，在第 1 個 epoch 的 `Progress` 裡取消 context
- **THEN** 回傳 `context.Canceled` 與只有 1 個 epoch 的結果，參數與 `Epochs: 1` 的 `Fit` 逐位元相同

#### Scenario: Stopping inside an epoch
- **WHEN** 一個 `Func` 層在第 2 個 batch 的 forward 中取消 context
- **THEN** 不會跑第 3 個 batch，回傳 `context.Canceled` 與沒有任何 epoch 的結果

#### Scenario: Stopping after an epoch's last batch
- **WHEN** 一個 `Func` 層在第 2 個 epoch 最後一個 batch 的 forward 中取消 context
- **THEN** 第 2 個 epoch 的 `Progress` 不會被呼叫，回傳 `context.Canceled` 與只有第 1 個 epoch 的結果

#### Scenario: A cancel after the last epoch finished
- **WHEN** 在最後一個 epoch 的 `Progress` 裡取消 context
- **THEN** 回傳全部 epoch 的結果，沒有錯誤

#### Scenario: A context that is already done
- **WHEN** 以已取消的 context 呼叫 `FitContext`
- **THEN** 參數不變，`Progress` 沒有被呼叫，回傳 `context.Canceled` 與空的結果

#### Scenario: A deadline
- **WHEN** 以已過期的 deadline 呼叫 `FitContext`
- **THEN** 回傳的錯誤符合 `errors.Is(err, context.DeadlineExceeded)`

#### Scenario: A nil context
- **WHEN** 以 nil context 呼叫 `FitContext`
- **THEN** 回傳錯誤，參數不變
