# Proposal: nn-edge-sum-all-cores

## Why

#379 step 2 measures whether a device kernel for `EdgeSum` is worth writing, and the operating contract measures a device only against an honest CPU. M19 made MatMul and Conv use every core before their device measurement for the same reason. `EdgeSum` and its reverse rule, merged in #387, run on one core.

## What Changes

- `EdgeSum`, and the two gradients `Tape.EdgeSum` computes, split their output elements across cores with `nn`'s existing `parallelFor`, choosing the worker count with `parallelWorkerCountForMACs` the way MatMul and Conv do. Each output element is still computed by one worker in the order the contract fixes, so no result changes by a single bit.
- A benchmark of one core against every core over a range of graph sizes, and the measured speedup recorded in `delivery-status.md`. It is the CPU baseline step 2's device measurement will be held against.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `nn-edge-sum`: the operation and its gradients use every core on large inputs, with results bit-identical to one core.

## Impact

- `nn/edge_sum.go`, `nn/autodiff_edge_sum.go`, tests and a benchmark.
- No API change and no result change. The CHANGELOG entry from #387 gains a note that large graphs use every core.
