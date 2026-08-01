# accel-runtime Specification

## Purpose
TBD - created by archiving change add-accel-runtime-capability. Update Purpose after archive.
## Requirements
### Requirement: Optional acceleration package boundary
The system SHALL expose GPGPU acceleration through an opt-in `insyra/accel` package family rather than through mandatory core-package dependencies.

#### Scenario: User imports core Insyra only
- **WHEN** a user imports `github.com/HazelnutParadise/insyra` without importing `insyra/accel`
- **THEN** core CPU workflows remain available without requiring GPU runtimes or native acceleration dependencies

### Requirement: Session-scoped acceleration runtime
The system SHALL define a session-scoped runtime surface that owns backend discovery, execution policy, and observable reports.

#### Scenario: User creates an accel session
- **WHEN** a user creates an acceleration session with `accel.Config`
- **THEN** the session owns device selection, memory budget policy, fallback policy, and execution reports for later accel operations

### Requirement: Typed execution surface
The system SHALL define `Dataset` and `Buffer` abstractions for GPU-eligible typed columnar execution.

#### Scenario: CPU data is prepared for accel execution
- **WHEN** a `DataTable` or `DataList` is projected into accel execution
- **THEN** the accel runtime uses typed datasets and buffers rather than reusing raw `[]any` storage directly

### Requirement: Observable execution result
The system SHALL define a public report surface for backend choice and execution outcomes.

#### Scenario: Accel operation completes
- **WHEN** an accel-eligible operation finishes
- **THEN** the runtime can return or expose a report containing selected backend, selected devices, and any fallback outcome

### Requirement: Session is safe for concurrent use
The system SHALL allow any number of goroutines to share one `Session`, with every public method safe to call concurrently with every other.

#### Scenario: Concurrent executions on one session
- **WHEN** multiple goroutines execute workloads through the same session at the same time
- **THEN** each execution's result is identical to what it would have produced running alone
- **AND** the race detector reports no data race

#### Scenario: Reads during execution
- **WHEN** one goroutine is executing a workload and another reads reports, devices, plans, or cache snapshots
- **THEN** the reader observes a consistent state, never a partially applied update

#### Scenario: Concurrent sessions share one device
- **WHEN** two sessions execute on the same host device at the same time
- **THEN** device submissions are serialized rather than interleaved

### Requirement: A process-shared session is available without construction
The system SHALL provide a session shared by the process, created on first use, that any caller can obtain without owning its lifetime.

#### Scenario: Two callers ask for the shared session
- **WHEN** two callers request the process-shared session
- **THEN** both receive the same session
- **AND** device discovery has run once, not once per caller

#### Scenario: Importing the package
- **WHEN** a program imports the accel package but never requests the shared session
- **THEN** no device is opened and no discovery runs

#### Scenario: No accelerator is present
- **WHEN** the shared session is requested on a host with no usable device
- **THEN** a usable session is still returned
- **AND** its report explains why acceleration is unavailable

#### Scenario: A caller closes the shared session
- **WHEN** a caller calls Close on the process-shared session
- **THEN** the call succeeds without error
- **AND** the session remains usable for every other caller

