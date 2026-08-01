## ADDED Requirements

### Requirement: Full GPU string kernels are a separate phase
The system SHALL treat full GPU string-kernel support as a separate Phase 2 capability rather than a Phase 1 runtime gate.

#### Scenario: Phase 1 runtime planning is reviewed
- **WHEN** the acceleration runtime, discovery, cache, and scheduler proposals are reviewed
- **THEN** full string-kernel parity is not required to declare Phase 1 planning complete

### Requirement: Phase 1 preserves encoded-string eligibility
The system SHALL preserve Phase 1 encoded-string transport and key-based eligibility while deferring full string-kernel parity.

#### Scenario: Workload uses string keys in Phase 1
- **WHEN** a Phase 1 accel workload uses string columns for transport or key-based behavior
- **THEN** the planning surface distinguishes that support from the deferred full string-kernel capability
