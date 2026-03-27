## ADDED Requirements

### Requirement: Acceleration mode configuration
The DSL SHALL allow acceleration mode configuration so scripts and REPL sessions can select the accel execution policy.

#### Scenario: Set acceleration mode
- **WHEN** a user runs `config accel.mode = strict-gpu`
- **THEN** the configured acceleration execution mode becomes `strict-gpu`

### Requirement: Acceleration inspection commands
The DSL and REPL SHALL expose acceleration inspection commands for device state and cache state.

#### Scenario: Show devices in REPL or script
- **WHEN** a user runs `show accel.devices`
- **THEN** the runtime prints discovered acceleration devices and backend summary

#### Scenario: Show cache in REPL or script
- **WHEN** a user runs `show accel.cache`
- **THEN** the runtime prints acceleration cache budget, residency summary, and related metrics
