# Proposal: nn-custom-optimizer

## Why

The last part of NN-2 ([#266](https://github.com/HazelnutParadise/insyra/issues/266)). `FitConfig.Optimizer` takes an `OptimizerSpec`, whose methods are unexported, so `Sequential.Fit` steps only with `SGD`, `SGDMomentum`, `Adam` and `AdamW`. Opening the interface would not have been enough, because nothing outside the package can write a parameter: `Parameter.Value()` returns a tensor whose data is private, and `Float32Data()` returns a copy.

Measured on ffff6a1d with an out-of-package test, a custom optimizer is possible today only in a hand-written loop, by running `SaveWeights` → `LoadSafeTensors` → compute → `LoadWeights` every step. SGD done that way matched `Fit` with `nn.SGD{Rate: 0.1}` bit for bit over 50 steps. But the route only works on a `Sequential`, cannot run inside `Fit`, and makes the caller transpose by hand: a Dense gradient is in the internal `[in,out]` layout, while `LoadWeights` takes torch's `[out,in]`.

The owner ruled on 2026-10-18 for option A on #266. `OptimizerSpec` stays sealed, for the reason `CustomLoss` gave: a sealed interface can gain a requirement as a new adapter field. Custom optimizers come in through an adapter, together with a public way to update a parameter from its gradient. PyTorch (`step()` reads `p.grad` and writes the parameter in place), Keras 3 (`update_step` with `assign`) and Gorgonia (`Solver.Step([]ValueGrad)`) all give an optimizer the value and the gradient and let it write the value.

## What Changes

- New `(*Parameter).Update(fn func(value, grad []float32)) error`:
  - It calls `fn` with the parameter's own float32 data and its most recent gradient, in the same layout, and keeps what `fn` writes.
  - `grad` is nil when the last reverse pass did not reach the parameter.
  - It returns an error for a nil parameter or a nil `fn`.
- New `nn.CustomOptimizer{Name, Step}`, which implements `OptimizerSpec`:
  - `Step func(params []*Parameter) error` is required. `Fit` refuses a `CustomOptimizer` without it before any batch.
  - `Fit` calls `Step` once per batch, after the reverse pass, with a copy of every parameter tracked on the model's tape, in creation order.
  - An error from `Step` stops `Fit`, wrapped with the optimizer's name. `Name` defaults to `CustomOptimizer`.
- The built-in optimizers are not touched.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `nn-training-frontdoor`: adds "Fit steps with an optimizer the caller writes".

## Impact

- `nn/fit.go`, `nn/autodiff.go` and the `nn` tests.
- `Docs/nn.md`, both changelogs, `api-review.md` (NN-2) and `delivery-status.md`.
- Nothing breaks, because the change only adds API.
