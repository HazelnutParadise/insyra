# Change: `insyra/dl` — run an ONNX MLP in pure Go, proved against onnxruntime

## Why
Models trained in PyTorch or TensorFlow cannot run in a Go service today without cgo bindings to onnxruntime. `ml` already writes ONNX; nothing in the ecosystem reads it back in pure Go with verified numerics. This change is the foundation slice of `insyra/dl`: a dtype-carrying tensor, a decoder for the ONNX format, a graph interpreter, and the MLP-level operator family — the smallest set that lets a real exported network load and predict, with every operator proved against `onnxruntime` rather than trusted.

Two decided constraints shape the internals beyond this change's own needs. Tensors carry a dtype even though f32 is the only implemented one, because the decided GGUF/LLM future runs on quantised types and the type system must not weld that door shut. And kernels are plain functions on tensors that the graph interpreter merely calls, because the future `llm` package hard-codes transformer architectures against the same kernels instead of interpreting graphs.

## What Changes
- New package `dl`: `Tensor` (row-major, N-D shape, dtype-carrying, f32 implemented), `LoadONNX(io.Reader)` returning a validated `*Model`, and `Model.Run(inputs map[string]*Tensor)` executing the graph in topological order
- The decoder treats the file as untrusted input: malformed bytes error, never panic; unsupported operators are collected and reported **all at once by name** at load time, never discovered mid-run
- MLP operator family as plain kernel functions: `Gemm`, `MatMul` (2-D), `Add`/`Sub`/`Mul` (with numpy-style broadcasting), `Relu`, `Sigmoid`, `Tanh`, `Softmax`, `Identity`, `Reshape`, `Flatten`, `Transpose`, `Cast` (float types only), `Constant`
- Per-operator verification through a generated one-op-graph harness: for each kernel, a Python helper builds a minimal `.onnx` containing only that operator, `onnxruntime` produces the reference output, and `dl` must match within f32 tolerance — gated through `internal/reftest` so strict mode fails when the reference is missing
- Whole-model verification: an MLP built with the Python `onnx` package (weights fixed, no torch dependency) round-trips with outputs compared to `onnxruntime`
- The repo's new-package contract in the same change: `Docs/dl.md`, rows in both READMEs' package tables, `Docs/README.md` index, `allpkgs` registration, changelog entries in both languages, `skills/insyra/` updated
- The `Reference Verification` workflow installs the Python `onnx` package (its gate will require it) and runs the `dl` parity tests

## Impact
- Affected specs: `dl-inference` (new)
- Affected code: new `dl/` package; `.github/workflows/reference-verification.yml`; docs, changelogs, skills per the new-package contract
- Additive; no existing package changes behaviour
