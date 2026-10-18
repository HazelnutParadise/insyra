# Design: nn-custom-optimizer

## An adapter over a write primitive

`OptimizerSpec` keeps its unexported methods. `CustomOptimizer` is a struct with a function field, the shape `CustomLoss` has. The function receives every parameter at once, as in PyTorch's `step()` and Gorgonia's `Solver.Step`, not one at a time as in Keras's `update_step`. Seeing every parameter at once is what an optimizer needs for anything global, such as clipping by the norm of all gradients.

`Parameter.Update` is the write primitive. It works the same inside `Fit` and in a loop written by hand, so a caller who wants a custom optimizer without `Fit` needs nothing more.

## Why a callback over the live slices

- **The value slice is the parameter's own data.** Writing into it is the update, with no copy back and no layout to get wrong, because `value` and `grad` share one layout (the one `Value()` and `Grad()` report).
- **`grad` is the gradient `Backward` published,** not a copy, as with `p.grad` in PyTorch. Copying both slices for every parameter on every batch would add an allocation per parameter per step to every custom optimizer.
- **Neither slice outlives `fn`.** The documentation says both are valid only while `fn` runs, the convention of `bufio.Scanner.Bytes`.
- **A missing gradient is `nil`, not zeros.** The built-in optimizers skip a parameter without a gradient, and a nil `grad` lets a custom one do the same. Weight decay, for example, should still apply to such a parameter, and zeros would hide that the parameter was not reached.

## Why the built-in numbers cannot move

`SGD`, `SGDMomentum`, `Adam` and `AdamW` still call the same tape methods, and neither new path is called from them. The MNIST convergence tests and the parity tests keep guarding them, and a new test checks that a `CustomOptimizer` reimplementing SGD matches `nn.SGD` bit for bit, in losses and in parameters.

## Precision and device paths

`ENG.md`'s precision contract concerns float32 results that may come from a device. Writing a parameter is a host write of float32 data. Device MatMul (`nn/device_matmul.go`) and `EdgeSum` keep no device copy of their inputs between calls: each call uploads, dispatches and reads back. A written parameter is therefore what the next call sees, and no cached device buffer can go stale.

## State

Optimizer state lives in `Step`'s closure. A `*Parameter` keeps its identity across batches and across `Fit` calls on one model, because `Fit` reuses the model's tape, so a map keyed by `*Parameter` holds a moment per parameter the way `self.state[p]` does in PyTorch. Nothing on `Parameter` stores custom state, which keeps the struct and `SaveWeights` unchanged.

## Verification

- A `CustomOptimizer` doing plain SGD through `Update` matches `nn.SGD` bit for bit: training losses, validation losses and every parameter.
- `Step` is called once per batch with exactly `model.Parameters()` in order. Clearing the slice it was given does not affect the next batch.
- Refusals, without a panic:
  - a missing `Step`, before any batch and with the parameters unchanged;
  - an empty `Name`, which reads `CustomOptimizer`;
  - a `Step` error, which is wrapped with the name;
  - `Update` on a nil parameter or with a nil function.
- `Update` hands over the gradient `Tape.Grad` reports, writes the value the next forward pass reads, and gives `nil` for a parameter the loss did not reach.
