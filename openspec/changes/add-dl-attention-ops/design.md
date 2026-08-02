## Context

The `dl` package already loads a small, validated ONNX graph and executes
float32 tensor kernels in pure Go. The existing MLP path has a two-dimensional
matrix multiply and trailing-dimension broadcasting, but a transformer encoder
also needs batched matrix products, normalization, attention-axis softmax, and
shape-control operators. The change must remain dependency-free at runtime and
must preserve the existing named-input, named-output graph model.

## Goals / Non-Goals

**Goals:**

- Extend the tensor layer with N-D `MatMul` and NumPy-style broadcasting over
  leading batch dimensions while retaining the 2-D fast path.
- Add the attention-family kernels and graph dispatch needed by one encoder
  block, including initializer-backed ONNX control inputs, multiple Split
  outputs, and boolean comparison tensors.
- Prove each new operator and one fixed-weight encoder block against
  `onnxruntime`, then provide a manual smoke path for local models.
- Keep the implementation as plain functions over `*Tensor` and keep public
  documentation, changelogs, and agent guidance synchronized.

**Non-Goals:**

- Supporting arbitrary ONNX graphs or a general transformer runtime.
- Adding a runtime dependency on Python, PyTorch, cgo, or an external ONNX
  engine.
- Shipping a real model fixture or making the manual smoke path part of CI.

## Decisions

1. **Reuse shared broadcast layout helpers.** Elementwise kernels, comparisons,
   `Expand`, and batched `MatMul` use the same aligned-shape and stride logic.
   This keeps the NumPy broadcast rule in one place. A separate implementation
   per operator would make rank and singleton-dimension behavior drift.

2. **Keep the 2-D matrix path separate.** The common 2-D case stays on its
   existing tight loop. Higher-rank inputs compute a broadcast batch shape and
   dispatch each batch matrix using aligned strides. Incompatible batch shapes
   fail before allocation and name both input shapes.

3. **Keep static graph controls static.** Attribute forms are read directly.
   `ReduceMean` axes, `Squeeze` axes, `Slice` starts/ends/axes/steps, and
   `Split` sizes may use initializer inputs, while runtime-computed control
   inputs are rejected with an error naming the node and control. The values
   are validated as integer tensors before being passed to the kernel.

4. **Consume only LayerNormalization's primary output.** The loader accepts
   ONNX's optional empty second and third output slots when only the first
   output is consumed. Named Mean or InvStdDev outputs fail clearly because
   the attention graph does not need those statistics.

5. **Use generated reference graphs for verification.** The parity helper
   builds one-op models with fixed values and deterministic weights, runs them
   through `onnxruntime`, and compares typed outputs. A fixed single-block
   encoder combines two-head self-attention, residual additions, GELU, and two
   LayerNormalizations without introducing a model fixture.

## Risks / Trade-offs

- **[Float32 implementation can diverge from a reference runtime]** → Keep
  arithmetic in the existing float32 contract, use float64 accumulation where
  the kernels already require it, and compare with the established float32
  tolerance in generated parity tests.
- **[Real ONNX models can use unsupported operators or dynamic shapes]** → The
  loader reports all unsupported operators together; the manual smoke test
  fails on that error and uses size `1` for dynamic dimensions instead of
  pretending to support an unknown shape.
- **[Control tensors can be malformed]** → Validate initializer control tensor
  dtypes and integer ranges at execution, with the node and control operation
  in the returned error; reject runtime-computed controls explicitly.

## Migration Plan

No data migration or public API migration is required. Existing 2-D MLP graphs
continue through the same dispatch path. Consumers that want to inspect a real
model can set `INSYRA_DL_REAL_MODEL` and run the documented manual test. If the
change is reverted, remove the new operator dispatch and documentation together
with the OpenSpec change; no persisted state is affected.

## Open Questions

None for this change. Broader ONNX coverage remains a separate follow-up
decision.
