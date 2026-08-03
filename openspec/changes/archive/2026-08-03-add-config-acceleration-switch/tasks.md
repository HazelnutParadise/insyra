# Tasks

## 1. The switch

- [x] 1.1 `config.go`: `SetAcceleration`/`GetAccelerationEnabled`, default enabled, matching the singleton's existing naming and locking conventions
- [x] 1.2 Call-time consult in dl's device-matmul dispatch, accel's DeviceMatMul/session gate, and knnbridge; env override keeps precedence

## 2. Proof

- [x] 2.1 Tests: Config off → CPU results byte-for-byte and hook/device not consulted (counting fake); env set with Config on → devices off; re-enable → device consulted again (fake hook, no GPU needed); hardware test unchanged and passing
- [x] 2.2 Full dl and accel suites, plus -race, pass

## 3. Sync

- [x] 3.1 Docs/dl.md and Docs/accel.md describe the two layers and precedence; changelog entries both languages; skills updated
