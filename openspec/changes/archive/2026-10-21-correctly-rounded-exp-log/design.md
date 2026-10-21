## Context

`nn.Tanh` is the existing pattern (`nn/tanh.go`). CORE-MATH publishes proven correctly rounded binary32 algorithms for `exp` and `log`; their error bounds are the starting point.

## Goals / Non-Goals

**Goals:** correctly rounded `exp` and `log` on host and device. **Non-Goals:** other elementary functions; float64.

## Decisions

- Host: float64 approximation + margin + exception table + `math/big` oracle (written here: `math/big` has no `Exp` or `Log`), as `Tanh` does.
- Device: multi-limb u32 fixed point with Ziv's test and a wider fallback; no 64-bit integers in WGSL.
- Exhaustive checks use the float64 fast path over all inputs and the oracle only inside the margin; device checks use the bounded-chunk seam.

## Risks / Trade-offs

- Long-running exhaustive tests, gated by an environment switch.
- Integer exp and log cost more per element than float ones.
