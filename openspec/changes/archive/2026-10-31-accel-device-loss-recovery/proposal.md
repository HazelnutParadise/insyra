# Proposal: accel-device-loss-recovery

## Why

On Windows a dispatch that runs about two seconds or more is stopped by the operating system and the Vulkan device is lost. Insyra opened its one device for the process and never replaced it, so every later call failed with `vkQueueSubmit failed: -4` until the program restarted (issue #571, reproduced on an NVIDIA GT 1030: a 2.9 s dispatch ran, the next submit failed, and 12 later tests failed). gogpu/wgpu reports that failure as plain text that does not wrap its own `ErrDeviceLost`. Whether a fresh device can be opened in the same process is up to the driver: one standalone gogpu program on the GT 1030 got one after releasing every object of the lost device, and every later attempt, there and in Insyra, got `vkCreateDevice` -3 for at least 13 s of retries.

## What Changes

- A submit or readback error that says the device was lost retires the device: every object made on it is released, buffers and bindings made on it are marked lost, and the next call that needs a device tries to open a new one, reporting no device if the driver refuses.
- A lost buffer or binding reports `accel.ErrDeviceLost` (fallback reason `device-lost`) wherever it is used; a kernel rebuilds its pipeline for the new device.
- Insyra's own operations return their CPU result.

## Capabilities

### Modified Capabilities

- `accel-gpu-execution`: behaviour after the device is lost.

## Impact

`accel/internal/wgpu/lost.go` and the sites that submit or read back, `accel/executor.go`, `accel/types.go`, `accel/nn_matmul.go`, tests, `Docs/accel.md`, both CHANGELOGs.
