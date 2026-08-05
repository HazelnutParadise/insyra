# accel-scheduler Delta: measure-device-saturation

## ADDED Requirements

### Requirement: Multi-device execution is gated on a measured saturation point
The system SHALL NOT dispatch one operation across multiple devices until a recorded measurement shows a workload shape where a single device's wall time scales with the sharded axis, and the recorded saturation point SHALL be the number that opens or parks that work.

#### Scenario: The saturation curve exists on record

- **WHEN** the gated saturation benchmark runs on device hardware
- **THEN** it records wall time per test-row rung, doubling per rung, with upload, dispatch, and readback included
- **AND** the first rung whose wall time is at least 1.8x the previous rung's is recorded as the saturation point, or its absence is recorded for the measured range

#### Scenario: The decision is written where planners look

- **WHEN** the measurement completes
- **THEN** `delivery-status.md` carries the curve and the resulting decision, and the `AGENTS.md` multi-GPU follow-up reflects the measured answer instead of the measurement's absence
