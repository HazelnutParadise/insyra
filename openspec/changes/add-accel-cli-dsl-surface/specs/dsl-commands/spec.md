## ADDED Requirements

### Requirement: Acceleration mode configuration
蝟餌絞 SHALL ?? acceleration mode ?賭誘嚗雿輻?銵?DSL ??script ?閮剖? accel ??嚗?

#### Scenario: Set acceleration mode
- **WHEN** 雿輻?銵?`config accel.mode = strict-gpu`
- **THEN** 蝟餌絞閮剖? acceleration execution mode ?? strict-gpu

### Requirement: Acceleration inspection commands
蝟餌絞 SHALL ?? DSL/REPL ?賭誘?? acceleration devices ??cache state嚗?

#### Scenario: Show devices in REPL or script
- **WHEN** 雿輻?銵?`show accel.devices`
- **THEN** 蝟餌絞憿舐內 discovered acceleration devices ??backend summary

#### Scenario: Show cache in REPL or script
- **WHEN** 雿輻?銵?`show accel.cache`
- **THEN** 蝟餌絞憿舐內 acceleration cache budget?esidency summary?elated metrics
