## 1. Kernel

- [x] 1.1 Add `tanh_rounded` to the fixed-point WGSL library and a `rounded-tanh` kernel with `wgpu.TanhRounded`
- [x] 1.2 Add `accel.RoundedTanh` on a resident `DeviceBuffer`

## 2. Verification

- [x] 2.1 `wgpu` test: a sample against a `math/big` oracle, including both analytic boundaries and the host's table inputs
- [x] 2.2 `nn` gated test: `accel.RoundedTanh` equals `nn.Tanh` on all 2^32 inputs (`INSYRA_NN_EXHAUSTIVE=1`)

## 3. Docs

- [x] 3.1 `Docs/accel.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`
- [x] 3.2 `delivery-status.md`: M37 done, next ticket
