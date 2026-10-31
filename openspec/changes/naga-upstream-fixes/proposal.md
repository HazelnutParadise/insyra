# Proposal: naga-upstream-fixes

## Why

gogpu/naga, the WGSL compiler behind the device backend, fixed two bugs Insyra reported and had been writing around. Before v0.19.1 a scalar `select()` used as an operand lost its parentheses in Metal (gogpu/naga#95). Before v0.20.4 `countLeadingZeros` compiled to the bit index of the highest set bit on Vulkan, DirectX 12 and OpenGL, and `countTrailingZeros(0)` gave `0xffffffff` (gogpu/naga#133). The workarounds and the warnings that tell callers to avoid these built-ins can go once the module requires the fixed version. The `-race` note in `Docs/nn.md` is also stale: the upstream Metal `checkptr` abort (gogpu/wgpu#280) was fixed in wgpu v0.30.31, and the module is on v0.34.5.

## What Changes

- `go.mod` requires gogpu/naga v0.20.4. gogpu/wgpu stays at v0.34.5, its newest release, and the `go` directive is unchanged.
- `ieee_decode` already computes the subnormal shift with `countLeadingZeros(fx) - 8u`; with the fixed compiler it is right on Vulkan, where subnormal inputs to the correctly rounded division, square root, exp and log were wrong.
- The exact-sum library no longer documents or tests a ban on `select()`.
- `Docs/nn.md` no longer says the device path is off under `-race`.
- A device test compares `countLeadingZeros`, `countTrailingZeros` and an operand `select()` with the host.

## Capabilities

### Modified Capabilities

- `accel-gpu-execution`: the WGSL built-ins a kernel may rely on.

## Impact

`go.mod`, `go.sum`, `exact_sum.go`, `exact_sum_test.go`, `wgsl_builtins_test.go`, `Docs/nn.md`, `ENG.md`, both CHANGELOGs. gogpu/wgpu#370 and #371 are still open upstream, so nothing here touches them.
