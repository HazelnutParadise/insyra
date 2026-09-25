# Proposal: correctly-rounded-tanh

## Why

M34 of #379. `ENG.md` defines a float32 result that may come from a device as the correctly rounded value of the exact operation, and M37 puts `tanh` on the device. `nn.Tanh` computes `float32(math.Tanh(float64(x)))`: `math.Tanh` has an error of a few float64 ulps, and rounding that to float32 is the correctly rounded `tanh(x)` except where the error carries the value across the midpoint between two float32 values. Those inputs also depend on the platform: Go compiles `math.Tanh` with fused multiply-adds on arm64, and on amd64 only when built with `GOAMD64=v3` or above, and uses assembly on s390x.

The gradient has a plain platform difference. `Tape.Tanh`'s reverse rule computes `upstream * (1 - y*y)`, which Go fuses on arm64 and not on amd64, so the same training step can produce different bits on the two.

## What Changes

- `nn.Tanh` returns the correctly rounded `tanh(x)` for every float32 `x`. This is IEEE 754's recommended correct rounding (§9.2), which CORE-MATH and RLIBM already deliver for binary32. It is computed by Ziv's method. The float64 `math.Tanh` result is accepted when it lies farther than a wide margin from every float32 midpoint, which proves its float32 rounding. Otherwise `tanh` is evaluated at 256 bits with `math/big` and rounded from there. `tanh` of a nonzero float32 is transcendental (Lindemann–Weierstrass), so it is never exactly a midpoint and the high-precision path always decides. Two ranges are settled analytically:
  - for |x| < 2^-13, `x − tanh(x) < x³/3` stays below half the gap to the next float32, so the result is `x`;
  - for |x| ≥ 9.5, `1 − tanh(|x|) < 2e^{-19}` stays below 2^-25, so the result is ±1.
- `Tape.Tanh`'s gradient is `RN(upstream · RN(1 − RN(y·y)))`, every step rounded and none fused. That is IEEE 754's reproducible evaluation (§11): a fixed sequence of correctly rounded basic operations.
- Verification: a gated exhaustive test compares `nn.Tanh` on all 2^32 inputs against an independent `math/big` oracle. The oracle uses a different formula from the implementation's high-precision path and is itself anchored to 50-digit values computed with Python's `decimal`. The test is run once against the old implementation, so the number of inputs that change is recorded.

## Capabilities

### New Capabilities

- `nn-correctly-rounded-functions`: elementwise float32 functions whose results are correctly rounded, and the reproducible evaluation of their gradients.

### Modified Capabilities

(none)

## Impact

- `nn/kernels.go` (`Tanh`), a new `nn/tanh.go`, `nn/autodiff.go` (`tanhVJP`), tests.
- `nn.Tanh` and `Tape.Tanh` are released. Their results change only for the inputs the exhaustive run finds, and only toward the correctly rounded value. The gradient changes only on platforms that fused it. Both CHANGELOGs record the numbers.
- `Docs/nn.md` says `Tanh` is correctly rounded.
