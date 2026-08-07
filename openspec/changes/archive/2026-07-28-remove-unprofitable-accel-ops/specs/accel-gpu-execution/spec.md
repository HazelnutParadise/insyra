## REMOVED Requirements

### Requirement: Squared distance from rows to query points
**Reason**: The result grows with rows times query points while the input grows with rows, so reading the answer back dominates. Superseded by taking the reduction on the device.
**Migration**: None required; the operation never appeared in a release. Callers wanting distances per row take the nearest query points instead.

### Requirement: Nearest query point per row
**Reason**: The answer is `f32`, and the callers it was built for compute in `float64`, so it changes their results. Superseded by the exact operation, which uses a device to narrow the field and settles the answer in `float64`.
**Migration**: None required; the operation never appeared in a release. Use the exact nearest operation, which returns the `float64` answer.

## ADDED Requirements

### Requirement: Only profitable operations are offered
The system SHALL NOT offer a device operation that measurement shows is slower than the host performing the same work with every core available.

#### Scenario: An operation is measured to lose
- **WHEN** an accelerated operation is measured against a host using all its cores and does not win
- **THEN** the operation is removed rather than left available
- **AND** the measurement is recorded so the decision can be revisited on other hardware
