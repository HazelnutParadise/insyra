## ADDED Requirements

### Requirement: Acceleration handler registration
The command registry SHALL register acceleration-related handlers so CLI and REPL entry points can share the same control surface.

#### Scenario: Registry dispatches accel handler
- **WHEN** an accel command is dispatched through `Registry.Dispatch`
- **THEN** the registry routes the request to the accel handler with the shared execution context

### Requirement: Acceleration execution report visibility
The acceleration handler SHALL expose selected backend, selected devices, and fallback outcome through the shared execution path.

#### Scenario: Accel-enabled command completes
- **WHEN** acceleration-enabled execution finishes
- **THEN** the handler can surface backend choice, selected devices, and fallback reason through the shared execution path
