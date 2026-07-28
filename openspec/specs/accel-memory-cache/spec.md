# accel-memory-cache Specification

## Purpose
TBD - created by archiving change add-accel-columnar-layout-cache. Update Purpose after archive.
## Requirements
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

### Requirement: Projection does not allocate per element
The system SHALL project a typed column without allocating memory proportional to the number of values beyond the output buffers themselves.

#### Scenario: A large numeric column is projected
- **WHEN** a numeric column of N values is projected
- **THEN** the number of allocations does not grow with N
- **AND** a benchmark records the allocation count so a regression is visible

### Requirement: Projection accepts the same values it accepts today
The system SHALL continue to project every value it currently projects, including values whose named type has a numeric underlying kind.

#### Scenario: A column holds a named numeric type
- **WHEN** a column holds values of a type declared as a named numeric type, such as `type Celsius float64`
- **THEN** the column projects to the same buffer type and values as before

#### Scenario: A column holds a type with no numeric representation
- **WHEN** a column holds a value that cannot be represented as a number, and the column is otherwise numeric
- **THEN** projection reports the same error it reports today, naming the offending index

### Requirement: Dataset fingerprint cost is proportional to column bytes
The system SHALL compute a dataset fingerprint in time proportional to the size of the column data, and SHALL NOT render column values to text in order to fingerprint them.

#### Scenario: A large numeric column is projected
- **WHEN** a numeric column is projected into a dataset
- **THEN** the fingerprint is derived from the binary representation of the values
- **AND** computing it does not allocate a decimal rendering of the column

#### Scenario: Fingerprint cost is measurable
- **WHEN** the repository is tested
- **THEN** a benchmark reports the cost of fingerprinting a large numeric column, so the figure is checkable rather than asserted

### Requirement: Fingerprints remain content-addressed over every value
The system SHALL include every value of every column in the fingerprint, and SHALL NOT substitute sampling, length-only, or identity-based shortcuts.

#### Scenario: Two datasets differ in a single value
- **WHEN** two datasets are identical except for one value in one column
- **THEN** their fingerprints differ

#### Scenario: The same data is projected twice
- **WHEN** the same column contents are projected twice within a session
- **THEN** both projections produce the same fingerprint
- **AND** the cache reports one resident entry rather than two

#### Scenario: Two string columns share their concatenated bytes
- **WHEN** two string columns hold the same bytes divided between values differently, such as `["ab", "c"]` and `["a", "bc"]`
- **THEN** their fingerprints differ

