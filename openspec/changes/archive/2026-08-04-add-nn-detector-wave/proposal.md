# Change: The detector wave — a real YOLO runs, Loop and NMS included

## Why

tiny-YOLOv3 (opset 11, a real published detector) is the measured bar
for M31. Its inventory: seven straightforward operators (LeakyRelu,
Exp, Ceil, Round, Tile, ReduceMin, plus Sigmoid/Slice already present)
and two structural ones — NonMaxSuppression, and Loop, the
interpreter's last control-flow gap. The four Loop bodies contain only
Add and Identity (tf2onnx counter loops), so the lift is the Loop
protocol itself: subgraph decoding, trip count and condition, loop-
carried dependencies, and scan outputs.

## What Changes

- Elementwise/reduction operators: LeakyRelu (alpha), Exp, Ceil,
  Round (half-to-even per ONNX), Tile, ReduceMin (axes forms like
  ReduceMean).
- NonMaxSuppression per the ONNX spec: batched, per-class,
  center_point_box attribute as the file uses it, max_output_boxes /
  iou_threshold / score_threshold as optional inputs, deterministic
  selection order, int64 [n,3] output.
- Loop: GraphProto attributes decode into nested subgraphs at load
  (validated like the top graph, unsupported body ops reported with
  the subgraph named); execution runs the body per iteration with
  trip-count and condition semantics, loop-carried values threaded,
  scan outputs stacked. Subgraph value scoping follows ONNX rules
  (outer values visible, body-local names shadowed).
- The gated real-model parity test gains tiny-yolov3-11.onnx: fixed
  deterministic image and shape inputs, all three outputs (boxes,
  scores, indices) compared against onnxruntime — exact for the int64
  indices, f32 tolerance for boxes and scores.
- One-op parity rows for every new operator; Loop gets dedicated
  synthetic-graph tests (counter loop, condition-terminated loop,
  scan output) plus its role in the real model.
- Docs, changelogs both languages, skills — same change.

## Non-Goals

- No If/Scan (nothing needs them yet), no Loop-body training, no
  ConvTranspose/TopK (still unneeded).

## Impact

- Affected specs: `nn-inference`
- Affected code: nn decoder (subgraph attributes), executor (Loop),
  kernels, parity harness, real-model test, docs, changelogs, skills.
