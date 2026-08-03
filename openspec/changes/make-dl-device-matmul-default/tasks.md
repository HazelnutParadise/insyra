# Tasks

## 1. Rewire

- [ ] 1.1 `accel`: exported device-matmul surface wrapping the internal wgpu kernel, carrying the fallback-reason reporting the bridge had
- [ ] 1.2 `dl`: init wiring (not under `race`), respecting `INSYRA_ACCEL_DISABLE_WGPU=1`; `RegisterDeviceMatMul(nil)` clears; floor and dispatch sites unchanged
- [ ] 1.3 Delete `accel/dlbridge`; move any test-package cycle in `accel/internal/wgpu` tests to an external test package

## 2. Proof

- [ ] 2.1 Hardware bit-equality test relocated to dl's own suite (default path, no tags), gated on `INSYRA_ACCEL_GPU_TESTS=1`
- [ ] 2.2 Switch tests: env var set → CPU results byte-for-byte; hook cleared → CPU; sub-floor and batched → device not consulted
- [ ] 2.3 Full dl and accel suites pass; `-race` run of dl passes without reaching the device

## 3. Sync

- [ ] 3.1 Docs/dl.md rewritten for default-on with the switch; Docs/dlbridge.md removed and the docs index updated; changelog Unreleased entries amended in both languages; skills updated; delivery-status decision entry
