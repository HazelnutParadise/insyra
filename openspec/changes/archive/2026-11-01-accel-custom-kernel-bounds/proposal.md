# Proposal: accel-custom-kernel-bounds

## Why

[#555](https://github.com/HazelnutParadise/insyra/issues/555): gogpu/wgpu v0.34.5 compiles WGSL for Vulkan with no bounds checks and does not enable `robustBufferAccess` (gogpu/wgpu#370, open), so a caller's kernel indexing past its buffer could overwrite another resident buffer or lose the device. gogpu/naga v0.20.4, which Insyra already requires, can add the checks, and wgpu accepts SPIR-V. The fix landed in #623 (dev) without a change; this records it.

## What Changes

- On Vulkan, a caller's kernel is compiled by Insyra with naga's bounds checks (buffer reads past the end give zero and writes are skipped, as on Metal; other indices are clamped, as WebGPU requires) and handed to the device as SPIR-V. Insyra's own kernels and the other backends keep their WGSL path.
- `gogpu/naga` becomes a direct dependency at the version wgpu uses.
