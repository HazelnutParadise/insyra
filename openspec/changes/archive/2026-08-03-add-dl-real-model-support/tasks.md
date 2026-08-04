# Tasks

## 1. Operators and interpreter

- [x] 1.1 `Clip` (opset-12 input form, initializer or runtime min/max) and `ConstantOfShape`
- [x] 1.2 Runtime shape tensors: shape-consuming operators (Reshape, Expand, Slice, Squeeze, Unsqueeze, Gather indices, Concat, Split, ReduceMean axes) accept runtime i64 inputs; validation moves to execution time, errors still name the node
- [x] 1.3 Whatever else the two graphs need to execute (attribute forms, Cast targets, broadcast edges) — done means both models run

## 2. Proof

- [x] 2.1 Gated real-model parity test behind `INSYRA_DL_REAL_MODELS_DIR`: both models, fixed inputs, dl vs onnxruntime within f32 tolerance; clean skip without the dir
- [x] 2.2 Unit tests for Clip, ConstantOfShape, and runtime-shape evaluation on small synthetic graphs (ungated)
- [x] 2.3 Existing suites pass unchanged

## 3. Sync

- [x] 3.1 Docs/dl.md operator table and real-model section; changelogs both languages; skills
