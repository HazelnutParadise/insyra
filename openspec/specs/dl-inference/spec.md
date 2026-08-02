# dl-inference Specification

## Purpose
TBD - created by archiving change add-dl-onnx-mlp-inference. Update Purpose after archive.
## Requirements
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

### Requirement: Batched matrix multiplication broadcasts like the standard
The system SHALL execute N-D matrix multiplication with batched leading dimensions, broadcasting batch shapes by the numpy rules, matching the reference runtime.

#### Scenario: Batch shapes broadcast

- **WHEN** two inputs with different but broadcast-compatible leading shapes are multiplied
- **THEN** the result matches `onnxruntime` element for element within single-precision tolerance

#### Scenario: Batch shapes are incompatible

- **WHEN** leading shapes cannot broadcast
- **THEN** the operation is refused naming both shapes

### Requirement: A transformer encoder runs
The system SHALL execute the attention operator family such that a self-attention encoder block — attention, feed-forward, layer normalisation — loads and runs end to end.

#### Scenario: An encoder block matches the reference

- **WHEN** a fixed-weight encoder block model is run by both `dl` and `onnxruntime` on the same inputs
- **THEN** outputs agree within single-precision tolerance

#### Scenario: Every new kernel is proved individually

- **WHEN** the one-op parity harness runs
- **THEN** each attention-family operator has generated-graph rows, including broadcast and axis-attribute cases

### Requirement: Convolution matches the reference across its attribute space
The system SHALL execute 2-D convolution with padding, strides, dilations and groups, matching the reference runtime across attribute combinations rather than only defaults.

#### Scenario: Attribute combinations are proved individually

- **WHEN** the one-op parity harness runs for convolution and pooling
- **THEN** generated cases cover asymmetric padding, non-unit strides, dilation, grouped and depthwise convolution, and pooling padding modes, each matching the reference within single-precision tolerance

#### Scenario: An image classifier runs

- **WHEN** a fixed-weight convolutional classifier is run by both runtimes on the same inputs
- **THEN** outputs agree within single-precision tolerance

### Requirement: Hot kernels parallelize without changing a single bit

`MatMul` (the 2-D fast path and the batched path) and `Conv` SHALL distribute
their independent output elements across CPU cores while preserving the serial
accumulation order within every output element, so that parallel and serial
execution produce bit-identical results. Inputs below a measured size
threshold SHALL take the serial path unchanged.

#### Scenario: Parallel MatMul is bit-identical to serial

- **WHEN** a MatMul large enough to cross the parallel threshold is computed
- **THEN** every output element SHALL equal the serial result exactly, not
  within a tolerance, and the existing one-op and whole-model parity suites
  SHALL pass unchanged

#### Scenario: Parallel Conv is bit-identical to serial

- **WHEN** a Conv large enough to cross the parallel threshold is computed
- **THEN** every output element SHALL equal the serial result exactly, and
  the fixed-weight CNN whole-model proof SHALL pass unchanged

#### Scenario: Small inputs avoid parallel overhead

- **WHEN** a MatMul or Conv is below the parallel size threshold
- **THEN** it SHALL execute on the serial path with no goroutine dispatch

### Requirement: Device matmul is measured before it is proposed

A production device kernel for `dl`'s matrix multiplication SHALL NOT be
proposed until a prototype has been measured against the all-core CPU
baseline at dl's measured hot shapes, with transfer, dispatch, and readback
included in the device's cost, and the observed numeric deviation from the
CPU result recorded per shape.

#### Scenario: The measurement includes the full device cost

- **WHEN** the prototype benchmark compares device and CPU matmul at a hot
  shape
- **THEN** the device time SHALL include upload, dispatch, and readback, and
  the CPU time SHALL be the all-core parallel path, not a single core

#### Scenario: The precision consequence is decided from observed numbers

- **WHEN** the prototype reports its results
- **THEN** it SHALL record the maximum absolute and ULP deviation between
  device and CPU outputs per shape, and the go/no-go decision SHALL name the
  deviation it accepted or refused

#### Scenario: A losing device closes the milestone negatively

- **WHEN** the measurement shows the device failing to beat the all-core
  baseline at every hot shape
- **THEN** the milestone SHALL be closed with the recorded numbers and no
  kernel SHALL be written

### Requirement: Large matmuls may run on a device, invisibly and exactly

`dl` SHALL expose a device-matmul hook that is nil by default, consulted only
for 2-D f32 matmuls at or above a measured MAC floor, and filled only by
blank-importing an opt-in bridge package. With the hook nil the CPU path SHALL
be byte-for-byte what it was before this change. With the hook active, device
results SHALL be asserted bit-equal to the CPU path on hardware, and any
device absence, error, or parity failure SHALL fall back to the CPU path
observably rather than change any answer.

#### Scenario: No bridge, no change

- **WHEN** a program uses `dl` without blank-importing the bridge
- **THEN** no device is discovered, no accel code is linked beyond the hook
  declaration, and every result is the existing CPU result

#### Scenario: Bridge active, results exact

- **WHEN** the bridge is blank-imported, a device is present, and a 2-D matmul
  at or above the measured floor executes
- **THEN** the result SHALL be bit-equal to the CPU path's result, asserted by
  a hardware test with exact equality

#### Scenario: Device trouble is a performance event

- **WHEN** the bridge is active but the device is missing, errors, or the
  platform fails the parity assertion
- **THEN** the matmul SHALL return the CPU result, the fallback SHALL be
  observable, and only strict GPU mode SHALL fail instead

#### Scenario: Below the floor and batched shapes stay on the CPU

- **WHEN** a matmul is batched or below the measured MAC floor
- **THEN** the hook SHALL NOT be consulted, because measurement refused those
  shapes

