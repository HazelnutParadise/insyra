## ADDED Requirements

### Requirement: Acceleration command group
蝟餌絞 SHALL ?? `accel` ?賭誘蝯∠?嚗? acceleration ?賭誘閮餃??device?ache?xecution mode ??嚗?

#### Scenario: List acceleration devices
- **WHEN** 雿輻?銵?`insyra accel devices`
- **THEN** 蝟餌絞憿舐內 acceleration backend?iscovered devices?hosen capabilities

#### Scenario: Show acceleration cache
- **WHEN** 雿輻?銵?`insyra accel cache`
- **THEN** 蝟餌絞憿舐內 cache budget?esidency summary?viction-related state

#### Scenario: Run with explicit acceleration mode
- **WHEN** 雿輻?銵?`insyra accel run --mode strict-gpu <command> ...`
- **THEN** 蝟餌絞?隞斗??舟? acceleration mode ?厲?狀?? fallback ??蝯???
