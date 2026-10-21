## Why

`SoftmaxCrossEntropy` needs `exp` and `log` whose result is the exact value rounded once to float32, the same on every platform. Go's `math.Exp` and `math.Log` are not correctly rounded for float32 results and differ by platform, and WGSL bounds them only loosely.

## What Changes

- `nn` gains correctly rounded float32 `exp` and `log`, built from IEEE `+ - * /` only, in the pattern of `nn.Tanh`: a deterministic float64 approximation, a margin above its exhaustively measured error, and a table of the inputs inside the margin with a `math/big` oracle behind the table.
- The same functions in integer arithmetic on the device (32-bit limbs), equal to the host on every input.

## Capabilities

### Modified Capabilities
- `nn-correctly-rounded-functions`: adds `exp` and `log`.

## Impact

- Code: `nn/` (new functions, tables, oracle), `accel/internal/wgpu/` (kernels). API: additions only. Docs and changelogs.
- Cost: exhaustive checks run behind an environment switch like `Tanh`'s; the float64 fast path covers all 2^32 inputs in minutes, the oracle only the margin.
