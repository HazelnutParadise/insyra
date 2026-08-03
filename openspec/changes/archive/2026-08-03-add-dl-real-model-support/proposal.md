# Change: Two real published checkpoints run in dl and match onnxruntime

## Why

Every whole-model proof so far is synthetic — fixed weights our own
fixtures wrote. The package's claims meet reality only when a model
somebody else published runs unmodified. Two were chosen to cover both
op families and inventoried: `mobilenetv2-12.onnx` (opset 12, 105 nodes)
and `all-MiniLM-L6-v2` (opset 14, 780 nodes, a real 6-layer BERT-class
encoder). The measured gaps:

- MobileNetV2 needs exactly one new operator: `Clip` (35 uses).
- MiniLM needs `ConstantOfShape` (1 use) and — the substantial one —
  **runtime-computed shape tensors**. Its graph computes shapes at
  runtime from dynamic batch/sequence dimensions through
  Shape→Gather→Unsqueeze→Concat→Reshape/Expand/Slice chains (58 Shape
  nodes). The interpreter currently refuses non-initializer control
  inputs on principle; the principle loses to reality here, and the
  refusal is replaced by evaluation: small integer tensors flow through
  the graph like any other value and shape-consuming operators read
  them at execution time.

## What Changes

- `Clip` operator (min/max as attributes opset<11 style is not needed;
  opset 12 carries them as optional inputs — support the input form,
  initializer or runtime) and `ConstantOfShape`.
- Shape-consuming operators — `Reshape`, `Expand`, `Slice`, `Squeeze`,
  `Unsqueeze`, `ReduceMean` axes, `Gather` indices, `Concat` of shape
  vectors, `Split` — accept runtime i64 tensors wherever they currently
  demand initializers. Validation moves from load time to execution
  time for those inputs, with errors still naming the node.
- Whatever small gaps execution of these two graphs reveals beyond the
  inventory (attribute forms, Cast targets, broadcasting edges) are in
  scope: the change is done when both models run, not when the listed
  ops exist.
- A gated real-model parity test: with `INSYRA_DL_REAL_MODELS_DIR` set
  (the operator keeps the two files in `~/.cache/insyra-dl-models`),
  each model runs on fixed deterministic inputs in both dl and
  `onnxruntime` (via the crosslang venv) and outputs match within f32
  tolerance. Without the env var or the files, the test skips cleanly.
  CI does not download models; this is an operator-run gate like the
  GPU suite.
- Docs (operator table, real-model section), changelogs both languages,
  skills — same change.

## Non-Goals

- No tokenizer (MiniLM inputs are raw token-id tensors).
- No f16/bf16 (M23), no quantized models, no external-data (>2GB)
  tensor support, no If/Loop control flow — neither model needs them.
- No performance work (M24 measures first).

## Impact

- Affected specs: `dl-inference`
- Affected code: dl kernels, decoder/interpreter (runtime shape-tensor
  evaluation), tests, docs, changelogs, skills.
