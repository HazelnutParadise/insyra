## ADDED Requirements

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
