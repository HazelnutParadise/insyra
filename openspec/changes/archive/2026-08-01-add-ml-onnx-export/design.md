## Context

The `ml` package now has fitted linear, logistic, and tree models plus fitted
preprocessing and pipelines. An exporter must preserve the fitted feature
order, preprocessing graph, class labels, tree routing, and model outputs
without fitting or evaluating anything again. ONNX's `ai.onnx.ml` domain
contains operators for the supported model families, but the repository does
not need an ONNX runtime or a C binding to write them.

The export path therefore builds an ordinary `ModelProto` in memory from the
already fitted values. The repository uses a small internal, wire-compatible
protobuf surface backed by `google.golang.org/protobuf/encoding/protowire`.
This keeps the schema subset explicit and avoids depending on an abandoned
high-level ONNX Go package.

## Goals / Non-Goals

**Goals:**

- Export supported linear, logistic, decision-tree, scaler, encoder, and
  fitted-pipeline models as standard ONNX graphs.
- Apply preprocessing and model nodes in one graph so a pipeline accepts raw
  observations.
- Preserve feature order, categorical encoding, missing-value routing, class
  labels, and logistic link behavior.
- Build completely before touching the caller's writer and report short writes.
- Verify files with an independent runtime when that runtime is available.
- Refuse unsupported models without writing a partial artifact.

**Non-Goals:**

- Importing ONNX or embedding an ONNX runtime.
- Exporting unfitted estimators, offset-dependent GLM models, KNN, KMeans, PCA,
  or the fitted imputer in this change.
- Guaranteeing bit identity after ONNX's `float32` exchange representation;
  comparison uses the format's documented precision tolerance.
- Inventing an ONNX equivalent for a model whose semantics are not expressible
  by the selected operators.

## Decisions

### Use one internal graph builder for direct models and pipelines

`onnxBuilder` owns graph inputs, nodes, initializers, outputs, and deterministic
node names. Direct models and fitted pipelines both create one builder, apply
their transforms, append the final predictor, and finish one graph with the
required ONNX and `ai.onnx.ml` opset imports.

This avoids separate serializers that could disagree about feature order or
output shapes. Writing a preprocessed model and a model file separately was
rejected because it would force consumers to reproduce the preprocessing
sequence and would fail the raw-observation pipeline requirement.

### Represent each feature as a one-dimensional input and concatenate a matrix

The graph starts with one input per fitted feature. Integer and string inputs
are cast or encoded as required, each feature is unsqueezed to a column, and
multiple columns are concatenated along axis one. This mirrors the table API's
named columns while presenting the predictor operators with the matrix shape
they expect.

Feature names are validated for non-empty uniqueness before any graph is built.
The feature order comes from the fitted model or pipeline, never from the
order of the table supplied at export time.

### Map supported models to `ai.onnx.ml` operators

- Linear regression uses `LinearRegressor` with coefficients, intercept, and a
  scalar output.
- Logistic regression uses `LinearClassifier`, class-label attributes, and the
  fitted logistic post-transform.
- Classification and regression trees use tree-ensemble nodes, including the
  learned missing/default branch and the categorical encoding needed before
  traversal.
- Root scalers and encoders use the corresponding ONNX preprocessing nodes.

Each mapping is emitted from the fitted result fields rather than by refitting
or calling `Predict`. Models with no faithful mapping return an error naming
the concrete model type.

### Refuse before writing

`ExportONNX` calls the builder first, marshals the complete payload, and only
then writes it. A nil writer, invalid feature schema, unsupported transformer,
unsupported model, or invalid fitted coefficients therefore leaves the writer
untouched. A short write is returned as `io.ErrShortWrite`.

This ordering is stricter than writing nodes incrementally, but it prevents a
caller from mistaking an incomplete file for a valid export and makes refusal
safe for files, buffers, and network writers alike.

### Verify through an independent runtime

Tests inspect the generated graph and run the export through `onnxruntime`
when the Python module is installed. The runtime compares predictions on the
same observations, including a raw-input pipeline case. If the dependency is
absent, the round-trip test skips explicitly and reports the missing runtime;
the skip is not treated as a successful interoperability result.

## Risks / Trade-offs

- **[Risk] ONNX schema details or operator defaults change]** → Keep field
  numbers and operator attributes in the internal schema surface, test explicit
  zero-valued attributes, and load the bytes in a standard runtime when
  available.
- **[Risk] `float32` exchange precision differs from the package's `float64`
  calculations]** → Compare within an explicit tolerance and document the
  precision boundary rather than claiming bit identity.
- **[Risk] A tree's categorical or missing branch is encoded incorrectly]** →
  Build preprocessing from the fitted tree schema, retain the learned branch
  attributes, and compare tree predictions on missing and categorical inputs.
- **[Risk] A pipeline capability wrapper hides the underlying model type]** →
  Unwrap all fitted pipeline capability wrappers before selecting the graph
  exporter, while keeping the pipeline's fitted steps in the graph.
- **[Risk] The independent runtime is unavailable in CI]** → Keep structural
  graph tests unconditional and mark only the external round-trip as skipped
  with the exact missing dependency.

## Migration Plan

This is additive and has no persisted-state migration. Callers can invoke
`ExportONNX` or the fitted model's `ExportONNX` method and continue using the
existing Go model. Import and unsupported-model handling remain unchanged.
Rollback removes the exporter and its documentation without changing fitting
or prediction behavior.

## Open Questions

ONNX import, offset-aware GLM export, imputer export, and support for additional
models require separate mappings and compatibility tests. They must not be
enabled by treating an unsupported model as a generic graph.
