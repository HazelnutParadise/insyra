# Tasks

## 1. Tensor
- [x] 1.1 `dl.Tensor`: row-major data, N-D shape, explicit dtype (f32 implemented; other dtypes refused by name), constructors and shape/stride accessors
- [x] 1.2 numpy-style broadcasting semantics for element-wise ops, with shape-mismatch errors naming both shapes

## 2. ONNX decoding
- [x] 2.1 Protobuf decoder for ModelProto/GraphProto/NodeProto/TensorProto using `protowire` (dependency already present); no import of `ml` internals
- [x] 2.2 Load-time validation: opset read, initializers materialised, unsupported operators collected and reported all at once by name; malformed bytes error, never panic (fuzz-adjacent test with truncated/corrupted fixtures)

## 3. Interpreter and kernels
- [x] 3.1 Topological execution over named values; input binding by name with shape/type validation
- [x] 3.2 Kernels as plain exported-or-internal functions: Gemm, MatMul, Add, Sub, Mul, Relu, Sigmoid, Tanh, Softmax, Identity, Reshape, Flatten, Transpose, Cast (float only), Constant
- [x] 3.3 Mid-graph shape incompatibilities error naming the node

## 4. Verification
- [x] 4.1 One-op parity harness: Python helper (using the `onnx` package) builds a single-operator model per kernel; `onnxruntime` produces reference outputs; `dl` must match within f32 tolerance; gated via `internal/reftest`
- [x] 4.2 Whole-model parity: an MLP constructed with the Python `onnx` package (fixed weights, no torch), run by both sides, outputs compared
- [x] 4.3 Malformed-input tests: truncated file, wrong magic, unsupported op list completeness, missing/misshapen inputs
- [x] 4.4 Batch invariance: running rows one at a time equals running them as one batch

## 5. The new-package contract (same change, not a follow-up)
- [x] 5.1 `Docs/dl.md` following an existing page's structure; row in `## Packages` of README.md AND `## 套件` of README_TW.md; `Docs/README.md` index
- [x] 5.2 Register in `allpkgs/allpkgs.go`
- [x] 5.3 `## Unreleased` entries in CHANGELOG.md and CHANGELOG_TW.md under a `### dl` heading
- [x] 5.4 `skills/insyra/SKILL.md` gains dl usage guidance
- [x] 5.5 `reference-verification.yml` installs the Python `onnx` package and runs the dl parity tests
