## ADDED Requirements

### Requirement: A loaded network is a protocol model
The system SHALL adapt a loaded ONNX model into the estimator protocol: features bound by column name, predictions returned as a data list, classifier adapters reporting classes and probabilities.

#### Scenario: A network is scored like any other model

- **WHEN** a caller binds a loaded model with feature names and passes a feature table
- **THEN** columns are matched by name in the bound order, missing features are refused naming the column, and extra columns are ignored

#### Scenario: A classifier adapter reports probabilities

- **WHEN** a bound classifier predicts
- **THEN** its class labels come from the caller-supplied class list, the label is the highest-probability class, and the probability table's columns follow the class order

#### Scenario: The adapter is checked for conformance

- **WHEN** the protocol conformance checks run against either adapter
- **THEN** they pass

### Requirement: What insyra writes, insyra reads
The system SHALL execute the `ai.onnx.ml` operator domain used by `ml`'s exporter, so every model family `ml` exports loads and runs in pure Go.

#### Scenario: An exported model reads back

- **WHEN** any model family `ml` exports is written to ONNX and loaded by `dl`
- **THEN** running it reproduces the original fitted model's own predictions within single-precision tolerance
- **AND** matches `onnxruntime` on the same file and inputs

#### Scenario: An exported pipeline reads back

- **WHEN** a fitted pipeline with supported preprocessing is exported and loaded
- **THEN** the whole graph — preprocessing and estimator — executes and matches both references
