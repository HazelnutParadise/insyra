# Tasks

- [x] Recognise a lost device from the sentinel and from the Vulkan, DirectX 12 and map error texts (`deviceLost`).
- [x] Retire the lost device: release every tracked object, mark buffers and bindings lost, reset the kernels, the parameter ring and the deferred releases, and forget the shared handle.
- [x] Report `ErrDeviceLost` and the reason `device-lost` from the resident path and the built-in operations.
- [x] Unit tests that need no GPU for the recognition and the bookkeeping.
- [ ] Hardware test on a real GPU (`INSYRA_ACCEL_DEVICE_LOSS_TESTS=1`): provoke a loss, see the error, recover on a new device.
- [x] `Docs/accel.md` and both CHANGELOGs.
- [ ] `openspec validate accel-device-loss-recovery --strict` passes (CI runs it).
