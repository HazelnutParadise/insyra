## ADDED Requirements

### Requirement: Acceleration handler registration
蝟餌絞 SHALL ?? acceleration-related handlers ?? Registry嚗? CLI?EPL??單?梁??剔? acceleration control surface嚗?

#### Scenario: Registry dispatches accel handler
- **WHEN** 雿輻?? `accel` ?賭誘?砍?璅?楝?勗 Registry.Dispatch
- **THEN** Registry 頝舐?券? acceleration handler ??shared execution context

### Requirement: Acceleration execution report visibility
蝟餌絞 SHALL ?? acceleration handler ?瑁? selected backend?elected devices?allback outcome ???唳?報?

#### Scenario: Accel-enabled command completes
- **WHEN** acceleration-enabled execution finishes
- **THEN** the handler can surface backend choice, selected devices, and fallback reason through the shared execution path
