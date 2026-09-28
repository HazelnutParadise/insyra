# Proposal: nn-edge-sum

## Why

#379: CoImNet's recurrent core is a fixed sparse edge list whose weights and node states change during training, and it must stay O(N+E). insyra's only way to express "each node sums its weighted incoming edges" today is a dense N×N `MatMul`, which is O(N²) in memory and work.

The owner decided on 2026-09-25 (`delivery-status.md`) that #379 is built as a general building block with its own reverse rule, not a CoImNet-specific recurrent step: the caller composes the step on the tape from existing operations. PyTorch and TensorFlow do the same with `index_add_`/`scatter_add_` and `unsorted_segment_sum`.

This change is the CPU operation and its gradient. The device kernel, resident state and state export are later changes, and the kernel only if measurement shows the device wins.

## What Changes

- `nn.NewEdgeTopology(nodes int, sources, targets []int) (*EdgeTopology, error)` builds an immutable edge list: `nodes` nodes, edge `e` running from `sources[e]` to `targets[e]`. It validates every index, copies the slices, and precomputes the edges grouped by target and by source.
- `nn.EdgeSum(topology, weights, values)` returns `out[..., t] = Σ weights[e]·values[..., sources[e]]` over the edges whose target is `t`. `weights` is float32 `[E]`; `values` is float32 `[N]` or `[B, N]`; the output has the shape of `values`. Work and memory are O(E + size of values).
- `Tape.EdgeSum` records the same operation with its reverse rule, giving gradients for `weights` and `values`.
- The summation order is part of the contract, so a later device kernel can be held to it bit for bit: each output sums its edges in ascending edge index, starting from zero, and every product is rounded to float32 before it is added (no fused multiply-add). The gradients follow the same rule: `values` gradients sum a source's edges in ascending edge index, and `weights` gradients sum over the batch in ascending batch index.

## Capabilities

### New Capabilities

- `nn-edge-sum`: a sparse edge-sum operation and its reverse rule on float32 tensors.

### Modified Capabilities

(none)

## Impact

- New code in `nn`: a topology type, the kernel, the tape operation, tests.
- `Docs/nn.md`, `skills/insyra/SKILL.md`, both CHANGELOGs under `` ### `ml` and `nn` ``.
- Additive. No existing signature or result changes.
