# Tasks

- [x] `accel/internal/wgpu/bounds.go`: `shaderModuleDescriptor` and `boundedSPIRV`; a `bounded` flag on caller kernels.
- [x] Tests: the bounded SPIR-V carries more code than the unchecked one; on a GPU, a kernel writing 64 floats into a 4-float buffer leaves 8 neighbouring buffers unchanged (M3/Metal and GT 1030/Vulkan, #623).
- [x] `Docs/accel.md` and both changelogs.
- [x] `openspec validate accel-custom-kernel-bounds --strict` passes.
