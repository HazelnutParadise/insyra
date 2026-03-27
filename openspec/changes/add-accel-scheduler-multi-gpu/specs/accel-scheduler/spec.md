## ADDED Requirements

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
