# Tasks

- [x] Recognise a lost device from the sentinel and from the Vulkan, DirectX 12 and map error texts (`deviceLost`).
- [x] Retire the lost device: mark buffers and bindings lost, reset the kernels, the parameter ring and the deferred releases, forget the shared handle, and release every object it made (shader modules and pipeline layouts included) before the next device is opened.
- [x] Report `ErrDeviceLost` and the reason `device-lost` from the resident path and the built-in operations, checking a lost buffer or binding before trying a new device.
- [x] Unit tests that need no GPU for the recognition and the bookkeeping.
- [x] Hardware test on an NVIDIA GT 1030 (`INSYRA_ACCEL_DEVICE_LOSS_TESTS=1`): a loss is provoked and reported, old buffers and bindings report it, and the next call reports `no-accelerator` because the driver refused a new device in the same process (`vkCreateDevice` -3 for at least 13 s of retries, in Insyra and in a standalone gogpu program).
- [x] `Docs/accel.md` and both CHANGELOGs.
- [x] `openspec validate accel-device-loss-recovery --strict` passes (CI, OpenSpec strict validation on the PR).
