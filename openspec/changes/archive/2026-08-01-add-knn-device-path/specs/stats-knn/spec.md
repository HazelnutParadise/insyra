## ADDED Requirements

### Requirement: KNN can be answered by an accelerator without depending on one
The system SHALL let a registered device searcher answer auto-algorithm KNN queries in batch, keep `stats` free of any accelerator dependency, and change nothing when no searcher is registered.

#### Scenario: No bridge is imported

- **WHEN** KNN runs without the bridge package imported
- **THEN** behaviour and dependencies are exactly as before

#### Scenario: The bridge is imported and the shape earns the device

- **WHEN** the bridge is imported, the algorithm is auto, k is within the shortlist budget and the per-row work clears the measured floor
- **THEN** the device answers every test row's k nearest in one batch
- **AND** the results equal the brute-force CPU results index for index, because the operation decides in float64 on the host

#### Scenario: An explicit algorithm is chosen

- **WHEN** the caller names brute force or a tree algorithm
- **THEN** the device is never consulted

#### Scenario: The device declines or misbehaves

- **WHEN** the runtime does not accelerate the call, or the registered searcher returns a malformed answer
- **THEN** the query falls back to the CPU path
- **AND** a malformed answer is rejected by shape validation rather than reaching the caller

### Requirement: The wiring direction is measured, not inferred from the transposed one
The system SHALL carry a benchmark in the direction the wiring actually runs — test rows as the dataset, training rows as the queries — because the transposed direction's arithmetic is identical but its device efficiency need not be.

#### Scenario: The benchmark runs on a device host

- **WHEN** the device benchmarks run
- **THEN** a true-direction KNN arm reports device against all-core CPU on the same shapes
