## ADDED Requirements

### Requirement: An exported network loads and predicts in pure Go
The system SHALL load an ONNX model from bytes and execute its graph on caller-supplied named inputs, in pure Go, returning named output tensors.

#### Scenario: A trained MLP is loaded and run

- **WHEN** a caller loads an `.onnx` file whose operators are all supported and supplies correctly-shaped inputs
- **THEN** the outputs match `onnxruntime`'s outputs for the same file and inputs within single-precision tolerance

#### Scenario: The file is malformed

- **WHEN** the bytes are not a valid ONNX model
- **THEN** loading returns an error and never panics, because a model file is untrusted input

#### Scenario: The model uses operators outside the supported set

- **WHEN** a model contains unsupported operators
- **THEN** loading fails with one error naming every unsupported operator, not merely the first

#### Scenario: Inputs do not match the model

- **WHEN** a required input is missing, misshapen, or of the wrong type
- **THEN** running returns an error naming the input and the mismatch

### Requirement: Every kernel is proved against the reference runtime
The system SHALL verify each supported operator against `onnxruntime` on generated single-operator graphs, and route the reference dependency through the shared toolchain gate.

#### Scenario: An operator kernel is tested

- **WHEN** the parity harness runs for any supported operator
- **THEN** a minimal one-operator model is generated, both runtimes execute it on the same inputs, and outputs must agree within single-precision tolerance

#### Scenario: The reference is absent

- **WHEN** Python with `onnx` and `onnxruntime` is unavailable
- **THEN** the harness reports through the shared reference-toolchain gate — skipping by default, failing under strict mode

### Requirement: Tensors carry their dtype and kernels stand alone
The system SHALL represent tensors with an explicit dtype even while f32 is the only implemented one, and SHALL expose kernels as plain functions independent of the graph interpreter.

#### Scenario: A kernel is called without a graph

- **WHEN** a caller invokes an operator function directly on tensors
- **THEN** it computes without any model or interpreter involved

#### Scenario: An unimplemented dtype is encountered

- **WHEN** a model or tensor declares a dtype the runtime does not implement
- **THEN** the operation is refused with an error naming the dtype, rather than silently reinterpreting the data
