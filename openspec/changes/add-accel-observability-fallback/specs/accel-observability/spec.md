## ADDED Requirements

### Requirement: Stable fallback reason codes
The system SHALL define stable reason codes for acceleration fallback outcomes.

#### Scenario: Workload falls back to CPU in automatic mode
- **WHEN** an accel-eligible request does not execute on acceleration
- **THEN** the runtime records a stable fallback reason code rather than only free-form text

### Requirement: Execution report includes backend choice
The system SHALL expose execution reports that include backend and device selection outcomes.

#### Scenario: User inspects an accel execution result
- **WHEN** a user or command surface inspects an accel execution report
- **THEN** the report includes selected backend, selected devices, and whether fallback occurred

### Requirement: Cache and device usage visibility
The system SHALL expose minimal cache and device usage metrics.

#### Scenario: User inspects accel state
- **WHEN** a user requests accel cache or device state
- **THEN** the runtime can report budget usage, resident data summary, and device participation summary

### Requirement: Strict and automatic mode report semantics
The system SHALL keep report semantics compatible across strict and automatic execution modes.

#### Scenario: Strict GPU mode cannot execute
- **WHEN** strict GPU mode is selected and acceleration cannot proceed
- **THEN** the runtime returns an error
- **AND** the failure surface still identifies the selected policy and the reason acceleration could not run
