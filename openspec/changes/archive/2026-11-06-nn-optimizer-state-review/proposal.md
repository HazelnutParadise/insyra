## Why

The review of the merged optimizer-state work (#673, #674) found that `Clear`'s behaviour depended on an earlier `SetClearOptions`, so a helper calling `Clear()` to clean up could silently keep gradients, against the settings rules in `AGENTS.md`; that `ClipGradNorm` counted frozen parameters (trainable `a=[3,4]` and frozen `b=[30,40]` gave a norm of 50.249 and cut `a`'s gradient tenfold); that a file from a newer release was reported as an unknown tensor; that `Float32` and `Float32Rounded` tapes refused each other's files although they run the same Adam; that the reader checked the same descriptors two or three times and the format carried a constant `P/dtype` and a step split into two limbs; that the docs asked callers to save dropout's random state, which nothing exposes; and that a failed first resident Adam step left its new moment buffers behind. None of it is released.

## What Changes

- **BREAKING (unreleased)**: `Tape.Clear(opts ...ClearOptions)`; `SetClearOptions` and `GetClearOptions` are removed. The options apply to that call; more than one makes every later `Param` and `Backward` fail.
- `ClipGradNorm` leaves frozen parameters out of the norm and does not scale their gradients.
- Optimizer-state format version 2: version and precision in the SafeTensors `__metadata__`, read first; `P/dtype` removed; `P/step` one I64 scalar. Precision is `exact` or `float32`, the latter shared by `Float32` and `Float32Rounded`. The header is validated once.
- `Docs/nn.md` says dropout's random numbers cannot be resumed instead of asking callers to save them.
- A failed first resident Adam step releases the moments it created.

## Impact

- `nn/autodiff.go`, `nn/autodiff_device.go`, `nn/optimizer_lifecycle.go`, `nn/optimizer_state.go`, `nn/optimizer_state_reader.go`, `nn/safetensors.go` (internal metadata support), `nn/adam_rounded.go`, `Docs/nn.md`, both changelogs.
