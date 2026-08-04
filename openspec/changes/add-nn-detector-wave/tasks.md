# Tasks

## 1. Operators

- [ ] 1.1 LeakyRelu, Exp, Ceil, Round (half-to-even), Tile, ReduceMin
- [ ] 1.2 NonMaxSuppression: batched per-class, optional threshold inputs, deterministic order, int64 [n,3] output

## 2. Loop

- [ ] 2.1 Decoder: GraphProto attributes become validated nested subgraphs; unsupported body ops reported naming the subgraph
- [ ] 2.2 Executor: trip-count and condition semantics, loop-carried threading, scan-output stacking, ONNX outer-scope visibility

## 3. Proof

- [ ] 3.1 One-op parity rows for the seven operators; synthetic Loop graphs (counter, condition, scan) vs onnxruntime
- [ ] 3.2 Gated real-model parity: tiny-yolov3-11 — indices exact, boxes and scores within f32 tolerance
- [ ] 3.3 Existing suites pass unchanged

## 4. Sync

- [ ] 4.1 Docs operator table and real-model list; changelogs both languages; skills
