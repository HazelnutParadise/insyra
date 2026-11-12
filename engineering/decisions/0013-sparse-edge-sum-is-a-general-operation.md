# 0013 — Issue #379 is a general sparse edge sum, not a recurrent-step operation

**Status:** accepted
**Date:** 2026-09-25

## Context
Issue #379 asked for a CoImNet-style recurrent step on the tape, and asked not to adopt a CoImNet-specific schema. Its reverse rule could join a tape only through the custom-operation hook of #375, so #375 went first.

## Decision
`nn` provides `EdgeSum`, a general sparse edge-sum operation with its own reverse rule. The caller composes a recurrent step on the tape from existing operations. Device results are bit-identical to the CPU reference; if that costs too much, the owner discusses it before any tolerance is accepted. A device kernel is written only if measurement shows the device wins (record 0003).

## Options rejected
- A CoImNet-specific recurrent-step operation. PyTorch and TensorFlow offer the same building blocks (sparse-dense products, gather plus segment or scatter sums) rather than a recurrent-step operation.
- Accept run-to-run differences on GPUs, as PyTorch and TensorFlow do by default. Insyra's device rule requires the device to change no numbers.

## Consequences
`EdgeSum` is defined exactly by record 0014, and its device path is recorded in record 0016.
