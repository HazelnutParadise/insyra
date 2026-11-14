## Why

Record 0029: a resident recurrent step on the M3 beats the host only at in-degree 100, because the device `tanh` takes about 0.17 µs per element in 128-bit fixed point (2.85 s for 2^24) and is nearly all of the step below that. `EdgeSum` alone is 3 to 7 times faster on the M3 GPU than on its 8 cores. A cheaper exact scheme was recorded as a follow-up; the owner chose to pursue it on 2026-11-14.

## What Changes

- `tanh_rounded` tries a 64-bit fixed-point path first, as the host's `tanhFloat32` tries float64: `2|x| = n ln2/64 + r`, a 64-entry table for `2^(-(n&63)/64)`, nine series terms for `e^-r`, and two Newton steps on the reciprocal of `1 + e^-2|x|`, all in unsigned integer arithmetic except the f32 estimates that start the reduction and the reciprocal, which the exact steps after them correct. Its error is at most about three units of 2^-58; an input within 64 units of a rounding midpoint returns to the 128-bit path. On a host model of the arithmetic 243 of the 135,790,592 positive inputs in range do.
- The results are unchanged: the device still equals `nn.Tanh` on all 2^32 inputs.
- Its 130 constants are appended to the fixed-point library's constant buffer.

## Impact

- `accel/internal/wgpu/fast_tanh.go` (new) and `fixed_math.go`, tests; both changelogs, `AGENTS.md` (the follow-up resolved), `engineering/` (the measurement). API: none.
