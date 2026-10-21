## Context

`device-resident-tape-step` keeps `SGD` on the device. This change adds the optimizer most training uses. It needs only operations whose correctly rounded form is cheap: division (integer long division of 24-bit significands) and square root (integer square root), plus the two-term exact sums the register already does.

## Goals / Non-Goals

**Goals:** `Adam` and `AdamW` on device tensors, bit-identical to the host in `PrecisionFloat32Rounded`.

**Non-Goals:** `exp` and `log` (`correctly-rounded-exp-log`), `SoftmaxCrossEntropy` (`device-softmax-cross-entropy`), changing the default tape.

## Decisions

- **Each Adam step is a fixed sequence**, as in the specs, with two-term sums rounded once and every division and square root correctly rounded. Torch's Adam differs only at ulp level (`lerp`, `sqrt(v)/sqrt(bc2)`), and the default tape keeps float64.
- **Host square root is `float32(math.Sqrt(float64(x)))`**, correctly rounded because float64 carries at least 2×24+2 bits. Host division is IEEE.
- **Scalars are host values.** `1-beta^t` is computed in `math/big` and rounded once, so no float64 `Pow` result is the contract.
- **Device division and square root are integer algorithms** over 32-bit limbs; WGSL has no 64-bit integers.

## Risks / Trade-offs

- The default tape's `AdamW` and this mode's differ by a few ulps per step; documented.
- Exhaustive device division and square root checks need the bounded-chunk seam.
