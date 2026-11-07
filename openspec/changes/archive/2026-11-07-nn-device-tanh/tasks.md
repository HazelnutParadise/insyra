## 1. Implementation

- [x] 1.1 `rounded-tanh-grad` kernel, `wgpu.TanhGradRounded` and `accel.RoundedTanhGrad`; warm it with the other built-in kernels
- [x] 1.2 `Tape.Tanh` runs resident operands on the device; the rounded host gradient makes every zero `+0`

## 2. Verification

- [x] 2.1 `wgpu`: the gradient kernel equals Go float32 steps on special values and random inputs
- [x] 2.2 `nn`: a two-step `MatMul`/`Add`/`Tanh`/`MatMul` tape on the device equals the host bit for bit; the gradient's edge cases equal the host's; the rounded mode's zero is `+0`

## 3. Docs

- [x] 3.1 `Docs/nn.md`, `Docs/accel.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`
- [x] 3.2 `delivery-status.md`: M38 progress
