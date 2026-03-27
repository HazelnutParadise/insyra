## ADDED Requirements

### Requirement: Typed columnar projection
The system SHALL project GPU-eligible data into typed columnar layouts before accel execution.

#### Scenario: Numeric and boolean columns become accel-eligible
- **WHEN** a `DataTable` or `DataList` is prepared for accel execution
- **THEN** numeric and boolean columns are represented as contiguous typed buffers
- **AND** nullability is represented through validity metadata rather than through raw `nil` values inside GPU buffers

### Requirement: Encoded string transport
The system SHALL support Phase 1 string eligibility through encoded columnar transport.

#### Scenario: String columns are included in an accel-eligible workload
- **WHEN** a workload contains string columns that are eligible for transport or key-based operations
- **THEN** the runtime represents them through UTF-8 values, offsets, and optional dictionary/index buffers rather than through arbitrary Go string containers

### Requirement: Device and shared-memory cache budgets
The system SHALL define memory budgets for both discrete and shared-memory devices.

#### Scenario: Cache budget is computed for a selected device
- **WHEN** the runtime computes a cache budget
- **THEN** discrete devices use a device-local budget policy
- **AND** shared-memory devices use a working-set policy that is explicitly documented as shared residency rather than discrete VRAM

### Requirement: Deterministic cache identity and eviction
The system SHALL use deterministic cache keys and eviction policy for accel buffers.

#### Scenario: Same data and operation are reused
- **WHEN** the same eligible dataset layout and operation lineage are executed again
- **THEN** the runtime can reuse resident buffers if the cache key matches
- **AND** if memory pressure requires eviction, the runtime applies a defined eviction policy rather than arbitrary buffer removal
