# Proposal: correctly-rounded-tanh

## Why

M34 of #379. `ENG.md` defines a float32 result that may come from a device as the correctly rounded value of the exact operation, and M37 puts `tanh` on the device. `nn.Tanh` computes `float32(math.Tanh(float64(x)))`: `math.Tanh` has an error of a few float64 ulps, and rounding that to float32 is the correctly rounded `tanh(x)` except where the error carries the value across the midpoint between two float32 values. Those inputs also depend on the platform: Go compiles `math.Tanh` with fused multiply-adds on arm64, and on amd64 only when built with `GOAMD64=v3` or above, and uses assembly on s390x.

The gradient has a plain platform difference. `Tape.Tanh`'s reverse rule computes `upstream * (1 - y*y)`, which Go fuses on arm64 and not on amd64, so the same training step can produce different bits on the two.

## What Changes

- `nn.Tanh` returns the correctly rounded `tanh(x)` for every float32 `x`. This is IEEE 754's recommended correct rounding (§9.2), which CORE-MATH and RLIBM already deliver for binary32, computed the way they compute it:
  - Two ranges are settled analytically. For |x| < 2^-13, `x − tanh(x) < x³/3` stays below half the gap to the next float32, so the result is `x`. For |x| ≥ 9.5, `1 − tanh(|x|) < 2e^{-19}` stays below 2^-25, so the result is ±1.
  - Between them, a float64 approximation of our own computes `-expm1(-2a)/(2+expm1(-2a))`, with Cody–Waite reduction and a Taylor series. It uses only IEEE 754 `+ − × ÷`, and an explicit conversion rounds every product that feeds an addition or subtraction. Go's specification forbids fusing a product across such a conversion, and gc performs no other floating-point fusion, so the approximation returns the same bits on every platform. A CI test pins those bits with a hash over every input between the analytic ranges. `math.Tanh` does not: it contracts multiply-add on arm64, picks an FMA path at run time on amd64, and is assembly on s390x.
  - Ziv's method decides the rounding. The approximation's float32 rounding is accepted when it lies more than `tanhSettleSteps` float64 steps from every float32 midpoint, a margin above the approximation's largest error. Because the approximation is identical everywhere, the exhaustive run measures that error for every platform at once.
  - The few inputs inside the margin are listed with their correctly rounded results, the way CORE-MATH lists its exceptional cases. A `math/big` evaluation stays behind the table. It starts at 128 bits and doubles until the result is clear of both midpoints, so an input the table misses after a change to the approximation still gets the right answer, as long as the changed approximation stays within the margin. `tanh` of a nonzero float32 is transcendental (Lindemann–Weierstrass), so it is never exactly a midpoint and the high-precision path always decides.
- `Tape.Tanh`'s gradient is `RN(upstream · RN(1 − RN(y·y)))`, every step rounded and none fused. That is IEEE 754's reproducible evaluation (§11): a fixed sequence of correctly rounded basic operations.
- Verification: a gated exhaustive test runs `nn.Tanh` on all 2^32 inputs.
  - The 271,581,184 inputs between the analytic ranges are compared with an independent `math/big` oracle. The oracle uses a different formula from the high-precision path and is anchored to values computed to 60 digits with Python's `decimal`.
  - The analytic ranges are compared with their proven values, and those proofs are checked against the oracle on samples.
  - The same run measures the approximation's largest error and lists the inputs inside the margin. It fails unless the table holds exactly those inputs.
  - It was run once against the old implementation, so the number of inputs that change is recorded.

## Capabilities

### New Capabilities

- `nn-correctly-rounded-functions`: elementwise float32 functions whose results are correctly rounded, and the reproducible evaluation of their gradients.

### Modified Capabilities

(none)

## Impact

- `nn/kernels.go` (`Tanh`), a new `nn/tanh.go` carrying the approximation, the margin and the table, `nn/autodiff.go` (`tanhVJP`), tests.
- `nn.Tanh` and `Tape.Tanh` are released. Their results change only for the inputs the exhaustive run finds, and only toward the correctly rounded value. The gradient changes only on platforms that fused it. Both CHANGELOGs record the numbers.
- `Docs/nn.md` says `Tanh` is correctly rounded.
