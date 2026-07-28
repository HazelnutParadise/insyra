## ADDED Requirements

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
