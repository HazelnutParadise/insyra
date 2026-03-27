## ADDED Requirements

### Requirement: Acceleration command group
The CLI SHALL expose an `accel` command group for acceleration device inspection, cache inspection, and execution-mode reporting.

#### Scenario: List acceleration devices
- **WHEN** a user runs `insyra accel devices`
- **THEN** the CLI reports discovered acceleration devices, backend names, probe source, and capability summary

#### Scenario: Show acceleration cache
- **WHEN** a user runs `insyra accel cache`
- **THEN** the CLI reports cache budget, resident buffers, resident bytes, and eviction-related state

#### Scenario: Run with explicit acceleration mode
- **WHEN** a user runs `insyra accel run --mode strict-gpu`
- **THEN** the CLI reports the selected acceleration mode, backend choice, shard-planning summary, and any fallback outcome
