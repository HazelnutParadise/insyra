## MODIFIED Requirements

### Requirement: A missing state file starts an empty environment
When an environment has no `state.json`, or an empty one, the system SHALL start it with no variables and SHALL NOT report an error.

#### Scenario: No state file
- **WHEN** the environment's `state.json` does not exist and a command is run
- **THEN** the command runs with no variables

#### Scenario: Empty state file
- **WHEN** the environment's `state.json` is empty, as `Create` leaves it for a moment, and a command is run
- **THEN** the command runs with no variables
