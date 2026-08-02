## ADDED Requirements

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
