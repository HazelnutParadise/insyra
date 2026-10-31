# Proposal: accel-vulkan-correctness

## Why

The first run of the device tests on a non-Apple GPU (NVIDIA GT 1030, Vulkan 1.4.312) found defects that Metal hides: queue writes to storage buffers were unordered with the next kernel or readback, so `Zero()` and uploads could read back stale words; discovery kept an NVIDIA card's CUDA entry, which has no executor, so every operation fell back to the CPU; and two hardware tests sized on an M3 sent a dispatch long enough for Windows to drop the device, after which every later submit failed. The wrong subnormal arithmetic found in the same run is fixed by `naga-upstream-fixes`.

## What Changes

- A write to a resident buffer waits for the write to land on every backend except Metal.
- When one card is discovered through two backends, discovery keeps the one that has an executor.
- The 100,000-query hardware tests scale their query count to a per-dispatch budget off Apple and skip when the discovered backend has no executor.
- The shortlist smoke test compares distances to a few ulps.

## Capabilities

### Modified Capabilities

- `accel-gpu-execution`: ordering of writes, and which backend discovery keeps.

## Impact

`accel/discovery.go`, `accel/internal/wgpu/resident.go`, `custom.go`, tests, `Docs/accel.md`, both CHANGELOGs.
