# Design: nn-custom-loss

## An adapter, not an open interface

`LossSpec` keeps its unexported methods. `CustomLoss` is a struct with function fields, the shape `ml.Estimator` already has (`ENG.md`: a pipeline step is a fit function, not a configured object). Configuration goes into the closure. A later requirement from `Fit`, such as a reduction mode, becomes a new field whose zero value keeps today's behaviour.

## What `Fit` checks, and where

- **Before any batch.** `validateFitConfig` already refuses a missing loss. It now asks a loss that can be misconfigured for its own error, through an unexported method only `CustomLoss` has. A `CustomLoss` without `Loss` is refused there, so a run cannot start and fail on its first batch. A nil `*CustomLoss` is caught by the existing nil check.
- **Target dtypes.** `validateFitTargetType` switches on the built-in losses and has nothing to say about a custom one. `Validate`, when set, is where a custom loss checks its targets; it runs where the built-ins' checks run, on every batch and on the validation set.
- **What `Loss` returns.** `Fit` reads `loss.data[0]` and calls `Backward`, which needs a float32 scalar. A nil tensor, another dtype or another shape would panic on the read or fail inside `Backward` with a message that does not name the loss, so `CustomLoss` checks the result first and names itself. A scalar computed outside the tape passes `Backward` but gives every parameter a gradient of zero, so a run would silently not train; `CustomLoss` refuses a result that no operation on the tape produced, the check `BackwardFrom` already makes. The validation loss is computed on a throwaway tape and is held to the same rule, so one function behaves the same in both places.

`Name` only labels errors, so it defaults rather than being required.

## Verification

- A `CustomLoss` whose `Loss` calls `tape.MSELoss` trains bit-identically to `nn.MSE{}` under one seed, validation included.
- A loss composed from tape operations (`MSELoss` scaled by a scalar through `tape.Mul`) trains: parameters move and the loss falls.
- `Validate` runs on training and validation batches, and its error comes back wrapped with the loss's name.
- Refusals, each without a panic and before any parameter changes where the spec says so: a missing `Loss`; a nil result; a non-scalar result; an int64 result; a scalar built outside the tape; an empty `Name` shown as `CustomLoss`.
