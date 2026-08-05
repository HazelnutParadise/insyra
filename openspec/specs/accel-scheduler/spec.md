# accel-scheduler Specification

## Purpose
TBD - created by archiving change add-accel-scheduler-multi-gpu. Update Purpose after archive.
## Requirements
### Requirement: Shardable workload policy
The system SHALL define a bounded set of shardable workload classes for v1 multi-device execution.

#### Scenario: Workload is not shardable
- **WHEN** an operation cannot be partitioned safely under the v1 scheduler rules
- **THEN** the runtime does not attempt heterogeneous multi-device execution for that operation

### Requirement: Weighted heterogeneous partitioning
The system SHALL partition shardable workloads according to device capability rather than equal-sized splits.

#### Scenario: Two selected devices have different throughput and memory capacity
- **WHEN** the scheduler builds a shard plan
- **THEN** it assigns more work to the stronger device if doing so improves the estimated execution plan

### Requirement: Deterministic merge policy
The system SHALL merge partial results deterministically.

#### Scenario: Shardable workload finishes on multiple devices
- **WHEN** per-device partial results are available
- **THEN** the runtime merges them in a deterministic way
- **AND** if no backend-specific merge path is defined, it uses the CPU merge path

### Requirement: Strict and automatic execution modes
The system SHALL distinguish strict GPU execution from automatic fallback execution.

#### Scenario: User selects strict GPU mode
- **WHEN** the workload cannot be executed on the selected acceleration path
- **THEN** the runtime returns an error instead of silently falling back to CPU

#### Scenario: User selects automatic mode
- **WHEN** the workload is unsupported or not profitable for acceleration
- **THEN** the runtime may fall back to CPU and records the fallback outcome for reporting


### Requirement: Multi-device execution is gated on a measured saturation point
The system SHALL NOT dispatch one operation across multiple devices until a recorded measurement shows a workload shape where a single device's wall time scales with the sharded axis, and the recorded saturation point SHALL be the number that opens or parks that work.

#### Scenario: The saturation curve exists on record

- **WHEN** the gated saturation benchmark runs on device hardware
- **THEN** it records wall time per test-row rung, doubling per rung, with upload, dispatch, and readback included
- **AND** the first rung whose wall time is at least 1.8x the previous rung's is recorded as the saturation point, or its absence is recorded for the measured range

#### Scenario: The decision is written where planners look

- **WHEN** the measurement completes
- **THEN** `delivery-status.md` carries the curve and the resulting decision, and the `AGENTS.md` multi-GPU follow-up reflects the measured answer instead of the measurement's absence
