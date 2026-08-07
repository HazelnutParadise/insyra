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

### Requirement: Shard plans execute across devices under an explicit strategy
The system SHALL execute a shard plan's assignments across the eligible devices under a strategy of `single`, `auto`, or `forced`; `auto` SHALL shard only when every assignment stays at or above the recorded saturation point; results SHALL be bit-identical to single-device execution under every strategy; and each assignment's placement SHALL be observable in the execution result.

#### Scenario: Auto below the floor stays single

- **WHEN** the workload is smaller than twice the recorded saturation point
- **THEN** the plan executes on one device exactly as before

#### Scenario: Auto above the floor shards within the floor

- **WHEN** the workload is large enough that every assignment can stay at or above the saturation point
- **THEN** assignments dispatch concurrently across eligible devices and merge deterministically

#### Scenario: Forced sharding complies and reports

- **WHEN** the caller selects the forced strategy on an eligible multi-device set
- **THEN** the plan shards regardless of the floor and the report shows each assignment's device

#### Scenario: Concurrent equals sequential equals brute force

- **WHEN** the same multi-assignment plan runs concurrently and assignment-by-assignment on one device
- **THEN** both outputs are byte-identical and equal the brute-force reference

#### Scenario: An assignment's device failure costs only its share

- **WHEN** one assignment's device fails mid-plan
- **THEN** that assignment's rows re-run on the CPU path, the other assignments' results stand, and the fallback is reported per assignment

#### Scenario: The multi-GPU wall clock is honestly absent

- **WHEN** the documentation describes multi-device execution
- **THEN** it states that correctness is verified on single-device hardware and that no multi-GPU wall-clock measurement exists yet, under the standing hardware-coverage follow-up
