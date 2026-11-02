## Why

PR #674 review found that valid zero-payload SafeTensors can carry large dimensions and reach unchecked matrix output allocation. `[MaxInt,0]` multiplied by `[0,2]` panics before the shape is checked, while huge empty rows or batches can trigger pointless work or batch-base allocation.

## What Changes

- Check matrix output element counts before device dispatch and use the existing checked zero-tensor constructor for CPU MatMul and Gemm.
- Convert runtime allocation-size panics into errors at the narrow float32 allocation boundary.
- Return empty and zero-inner MatMul results without row loops or batch-base allocation, preserving vector/scalar shapes and Gemm arithmetic/bias validation.
- Preserve valid empty SafeTensors, existing public APIs and all valid numerical results on dev and 0.4.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `nn-inference`: matrix products reject unrepresentable outputs and handle empty dimensions without unnecessary allocation or iteration.

## Impact

`nn/kernels.go`, `nn/matmul_rounded.go`, `nn/tensor.go`, regression tests and nn documentation/changelogs. No dependency, public signature or precision change. Physical-memory exhaustion for otherwise representable outputs remains outside this size-validation fix.
