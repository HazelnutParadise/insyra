# accel-observability Specification

## Purpose
TBD - created by archiving change add-accel-observability-fallback. Update Purpose after archive.
## Requirements
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

### Requirement: Acceleration announces itself once per session
The system SHALL emit one info-level log line on a session's first accelerated execution naming the device, backend, mode, and shard strategy; SHALL emit one info-level line on the first fallback that occurs where a device could have run, naming the fallback reason; SHALL emit per-execution detail at debug level only; and SHALL NOT grow log volume with call count at info level.

#### Scenario: First device use logs once

- **WHEN** a session executes many accelerated operations
- **THEN** exactly one info line announces the device, backend, mode, and strategy

#### Scenario: First qualifying fallback logs once with its reason

- **WHEN** a session that could use a device falls back to the CPU repeatedly
- **THEN** exactly one info line names the fallback reason

#### Scenario: Detail is available at debug level

- **WHEN** the log level includes debug
- **THEN** each execution logs its operation, rows, chunk count, and per-assignment placement

#### Scenario: Log levels govern acceleration like everything else

- **WHEN** the root configuration silences info logging
- **THEN** acceleration writes nothing at info level

#### Scenario: The acceleration switch logs its own transitions

- **WHEN** `Config.SetAcceleration` changes the acceleration state
- **THEN** one info line names the new state
- **AND** calling it again with the same value writes nothing
