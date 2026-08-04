## Context

The `nn` ONNX reader currently validates one flat graph and the executor passes
one value map to each node. The detector change crosses that decoder/executor
boundary: ONNX `GRAPH` attributes carry Loop bodies, and those bodies need
their own validated nodes, inputs, outputs, and initializers while retaining
visibility of values from the enclosing graph. The new numeric operators also
need to follow the existing plain-function kernel and dispatch conventions.

## Goals / Non-Goals

**Goals:**

- Decode and validate nested GraphProto attributes at load time, including
  named errors for unsupported operators inside a body.
- Execute ONNX Loop trip-count, condition, loop-carried, and scan-output
  semantics with child scopes and correct zero-iteration shapes.
- Add the measured detector operators and verify them against `onnxruntime`
  one operator at a time, synthetically for Loop, and on tiny-YOLOv3.

**Non-Goals:**

- Add `If`, `Scan`, or a general control-flow framework beyond the Loop
  protocol required by the target model.
- Add dependencies, change public tensor layout, or reinterpret existing
  `DataList`/training APIs as ONNX kernels.

## Decisions

- **Represent subgraphs as validated `modelGraph` values.** The protobuf
  decoder stores GRAPH and GRAPHS attributes as the same recursive graph
  structure used by the top-level decoder. Model construction validates each
  graph with the existing operator/output/input checks and materialises local
  initializers. This keeps unsupported-body failures at load time and avoids a
  second decoder or a runtime-only validation path.
- **Keep execution scopes as layered maps.** A Loop body receives a fresh map
  containing copied loop inputs and local initializers, with lookup falling
  back to the parent map. Body-produced names are written only to the child
  map, so local names shadow outer values without mutating the outer graph.
  The body is executed through the same dependency-order runner as the main
  graph. This is simpler and safer than merging maps in place, while still
  matching ONNX lexical scoping.
- **Implement Loop as an explicit executor path.** The node reads optional
  trip-count and condition inputs, binds body inputs as iteration number,
  condition, and loop-carried values, then threads body outputs and stores
  scan outputs. Scan tensors are stacked on axis zero; zero iterations return
  empty leading dimensions derived from the body scan output declarations or
  runtime initial values where necessary.
- **Implement NMS with stable score ordering and exact scalar controls.** The
  kernel handles both corner and center box encodings, processes each batch and
  class independently, and returns int64 `(batch, class, box)` rows. Runtime
  control inputs take precedence over defaults and are validated as scalar
  int64/float32 tensors.
- **Preserve the existing parity harness.** New generated one-op and synthetic
  models remain in `onnx_parity.py`; Go keeps JSON results on stdout and helper
  diagnostics on stderr. The real-model gate uses the existing deterministic
  feed plumbing and adds exact index comparison for YOLO.

## Risks / Trade-offs

- [Risk] Empty Loop scans can lose element shape information → Mitigation:
  validate body output declarations and construct zero-leading-dimension
  tensors from their declared/runtime shapes, with explicit errors when a
  required shape cannot be inferred.
- [Risk] NMS ordering or threshold defaults differ across runtimes →
  Mitigation: encode the ONNX defaults and stable tie handling directly, then
  retain exact index parity in both one-op and YOLO tests.
- [Risk] Nested graph validation could accidentally permit top-level-only
  assumptions → Mitigation: run the same operator support and output-arity
  validation recursively, while reporting the graph name in every body error.

## Migration Plan

No data migration or external deployment is required. Add the recursive model
representation and kernels, extend parity fixtures and documentation, run the
scoped `nn` suite and the gated real-model test, then archive the OpenSpec
change after verification.

## Open Questions

None for the measured target. `If` and `Scan` remain intentionally outside
this change.
