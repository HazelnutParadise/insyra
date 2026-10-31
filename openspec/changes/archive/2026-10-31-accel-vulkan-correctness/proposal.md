# Proposal: accel-vulkan-correctness

## Why

The first run of the device tests on a non-Apple GPU (NVIDIA GT 1030, Vulkan 1.4.312) found three defects that Metal hides: the WGSL `countLeadingZeros` compiled to a bit index, so correctly rounded division, square root, exp and log were wrong for subnormal inputs; queue writes to storage buffers were unordered with the next kernel or readback, so `Zero()` and uploads could read back stale words; and two hardware tests sized on an M3 sent a dispatch long enough for Windows to drop the device, after which every later submit failed.

## What Changes

- `ieee_decode` computes the shift with `firstLeadingBit`, not `countLeadingZeros`.
- A write to a resident buffer waits for the write to land on every backend except Metal.
- The 100,000-query hardware tests scale their query count to a per-dispatch budget off Apple and skip when the discovered backend has no executor.
- The shortlist smoke test compares distances to a few ulps.

## Capabilities

### Modified Capabilities

- `accel-gpu-execution`: results on Vulkan, and ordering of writes.

## Impact

`accel/internal/wgpu/ieee.go`, `resident.go`, `custom.go`, tests, `Docs/accel.md`, both CHANGELOGs.
