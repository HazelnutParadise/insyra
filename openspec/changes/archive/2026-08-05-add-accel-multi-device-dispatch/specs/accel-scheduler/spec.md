# accel-scheduler Delta: add-accel-multi-device-dispatch

## ADDED Requirements

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
