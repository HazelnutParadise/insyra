# Proposal: exact-edge-sum

## Why

M33 of #379. `ENG.md` now defines a float32 result that may come from a device as the correctly rounded value of the exact operation. `EdgeSum` does not meet that definition yet: it adds each node's products in ascending edge index and rounds after every product, so its result depends on that order. A device can reproduce that order only by giving each node to one thread and adding sequentially in floating point, and WGSL does not promise to keep a floating-point loop's order or rounding (it permits reassociation, contraction and flush-to-zero).

An exact sum rounded once has none of these problems. It does not depend on the order of the terms, so any split of the work across cores, threads or devices returns the same bits; it is the most accurate result float32 can hold; and a device can compute it in integers, which WGSL does specify exactly. This is the Kulisch long accumulator, the technique ReproBLAS and ExBLAS use for reproducible sums, the latter on GPUs.

`EdgeSum` is unreleased (#387, #388), so its result definition can still change.

## What Changes

- An exact accumulator in `nn`: it adds float32 products as exact integers in a fixed-point register wide enough for every finite product (32-bit digits, carries deferred), and rounds the total once to nearest-even float32, handling subnormal results and overflow to infinity by the same rule.
- Special values are defined: a NaN operand, `0·∞`, or infinities of both signs make the result NaN; otherwise an infinite product makes it that infinity; a total that is exactly zero is `+0`, as when adding into a zeroed output.
- `EdgeSum`, and both gradients of `Tape.EdgeSum`, use the accumulator. Each output is the correctly rounded exact sum of its products, independent of edge order, batch order and worker count.
- The previous fixed-order requirement is removed. Tests compare against a `math/big` oracle rather than a reference loop.
- The CPU cost against the fixed-order version is measured with `BenchmarkEdgeSum` and recorded in `delivery-status.md`.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `nn-edge-sum`: each output is the correctly rounded exact sum of its products; the fixed summation order is removed.

## Impact

- `nn/edge_sum.go`, `nn/autodiff_edge_sum.go`, a new accumulator file and tests.
- `Docs/nn.md`, both CHANGELOGs (the unreleased `EdgeSum` entry is rewritten), `delivery-status.md`.
- The prototype kernels in `accel/internal/wgpu`'s tests implement the removed fixed order. They stay as the record of `measure-edge-sum-device`; `TestEdgeSumDeviceTiming` will report mismatches against `nn.EdgeSum` until M36 replaces them.
