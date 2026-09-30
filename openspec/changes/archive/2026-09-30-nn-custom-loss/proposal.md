# Proposal: nn-custom-loss

## Why

The second half of NN-2 ([#266](https://github.com/HazelnutParadise/insyra/issues/266)): `FitConfig.Loss` takes a `LossSpec`, whose methods are unexported, so `Sequential.Fit` trains only with the three built-in losses. A caller who wants any other objective, a Huber loss or a weighted MSE, has to give up `Fit` and write the whole training loop on the tape by hand, although the tape can already compute that objective.

The owner chose on 2026-09-30, on #266, to keep `LossSpec` sealed and add an adapter that implements it from a function, rather than exporting the interface's methods. A sealed interface can gain a requirement later as a new adapter field whose zero value keeps today's behaviour; an exported method set cannot grow without breaking every implementation. Going from the adapter to an open interface later breaks nobody; going back would.

Custom optimizers stay out: a parameter's value cannot be written from outside the package, so they need a way to update a parameter first.

## What Changes

- New `nn.CustomLoss{Name, Loss, Validate}`, which implements `LossSpec`:
  - `Loss func(tape *Tape, prediction, target *Tensor) (*Tensor, error)` is required. It returns the batch's loss as a float32 scalar computed with operations on `tape`, so the reverse pass reaches the parameters.
  - `Validate func(prediction, target *Tensor) error` is optional. When set, `Fit` calls it before `Loss` on every batch and on the validation set, the way the built-in losses check their targets.
  - `Name` is optional. It labels the loss in `Fit`'s error messages; empty means `CustomLoss`.
- `Fit` refuses, before any batch runs, a `CustomLoss` without a `Loss` function. When `Loss` returns something other than a float32 scalar, or a tensor no operation on the tape produced, `Fit` returns an error naming the loss instead of panicking or training on a gradient of zero.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `nn-training-frontdoor`: adds "Fit trains with a loss the caller computes on the tape".

## Impact

- `nn/fit.go` and the `nn` tests.
- `Docs/nn.md`, both changelogs, `api-review.md` (NN-2), `delivery-status.md`.
