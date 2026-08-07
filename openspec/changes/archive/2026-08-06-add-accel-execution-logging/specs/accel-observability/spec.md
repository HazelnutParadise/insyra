# accel-observability Delta: add-accel-execution-logging

## ADDED Requirements

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
