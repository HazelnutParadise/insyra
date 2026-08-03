# Change: Training round-trips to disk — SafeTensors out, ONNX out

## Why

`nn` trains but cannot persist: SafeTensors only reads, and nothing
exports a trained network into the ONNX format `nn`'s own inference
half runs. Training that cannot leave memory is a demo, not a
capability. This change closes the loop both ways (M27): weights save
to SafeTensors that torch itself can read back, and a trained
Sequential exports to an ONNX file that both `nn` and `onnxruntime`
run with matching outputs.

## What Changes

- `SaveSafeTensors(w io.Writer, tensors map[string]*Tensor) error`:
  writes F32, I64, and BOOL tensors (the native dtypes) with sorted
  tensor names for deterministic bytes, contiguous data-region layout,
  and the same validation discipline in reverse. What it writes,
  `LoadSafeTensors` reads back identically — and so does the Python
  `safetensors` library, verified through the gate, including torch
  loading our file and reproducing our `Predict` output.
- `Sequential.SaveWeights(w io.Writer) error`: the torch-convention
  named parameters (plus BatchNorm running statistics) through
  `SaveSafeTensors` — the inverse of `LoadWeights`, with the Linear
  transpose applied so torch reads our file as a normal state dict.
- `Sequential.ExportONNX(w io.Writer) error`: builds an inference
  graph from the layer stack — Dense→Gemm, activations, Conv2D→Conv,
  BatchNorm2D→inference BatchNormalization carrying the trained
  running statistics, pooling, Flatten, LayerNorm, Dropout omitted
  (inference identity) — writing via the protowire patterns
  `ml/onnx_export.go` established. Layers without an ONNX mapping
  (Func, Embedding for now) refuse with the layer named.
- The round-trip proof: train a small MLP and a small CNN briefly,
  export, then (a) `nn.LoadONNX` runs the export and matches
  `Predict` exactly on fixed inputs, (b) `onnxruntime` matches within
  f32 tolerance through the gate, (c) torch loads the safetensors and
  matches `Predict`.
- Docs, changelogs both languages, skills — same change.

## Non-Goals

- No Embedding/Func ONNX mapping yet (refused by name), no training
  checkpoints beyond weights (optimizer state stays in memory), no
  external-data ONNX, no quantized writing.

## Impact

- Affected specs: `nn-training`
- Affected code: nn safetensors writer, Sequential save/export, an
  ONNX graph writer for nn, fixtures, docs, changelogs, skills.
