## Context

SafeTensors correctly represents zero-sized dimensions. Rejecting valid empty tensors would hide unchecked allocation in their consumers and break the storage format. Default 2D MatMul and Gemm multiply output dimensions inside make before validating shape. Rounded and batched paths already use the zero constructor but still iterate huge empty dimensions or allocate batch offsets.

## Goals / Non-Goals

**Goals:** refuse unrepresentable matrix outputs before dispatch/allocation; preserve valid empty shapes and numerical results; avoid work for empty MatMul results; cover direct kernels and tape routes on both maintained lines.

**Non-Goals:** new APIs, arbitrary dimension/RAM limits, changed floating arithmetic or recovering fatal physical-memory exhaustion.

## Decisions

- Validate 2D output element count before the optional device hook; empty/zero-inner products return through the existing zero constructor. Use the same constructor for the CPU worker path and Gemm instead of raw multiplied make lengths.
- Keep makeLayout's shape/stride checks. Put a narrow recover around only float32 slice allocation so runtime allocation-size panics become errors. Do not copy the runtime's platform-dependent maxAlloc or impose maxInt/4 as a new 32-bit policy.
- Normalize batched vector/scalar output metadata before an empty/zero-inner early return. Skip batch-base arrays and row loops for such products. Rounded MatMul uses the same early-return rule.
- Gemm skips its outer loop only for empty output columns. Nonempty zero-inner results still compute Alpha*0, preserving negative zero, infinity/NaN behavior, and C broadcasting still runs.
- Independent Astra API/security consultation confirmed the concrete sibling paths and no new owner decision is needed within the authorized review fix.

## Risks / Trade-offs

- Empty metadata differs between vectors/matrices → assert exact scalar/vector/batch shapes and zero bits.
- Broad panic recovery hides errors → confine it to a single allocation expression, with count in the returned error.
- Physical OOM can be fatal → make no recovery or resource-budget promise for otherwise valid output sizes.
- Huge-empty regression tests can hang on old code → isolate those cases in a bounded subprocess.
