## ADDED Requirements

### Requirement: An unreadable state file stops the command
When an environment's `state.json` exists but cannot be read, the system SHALL stop before running the command and SHALL report an error naming the file and the cause, and SHALL leave the file unchanged.

#### Scenario: Truncated state file
- **WHEN** `state.json` is cut off mid-object and a command is run on that environment
- **THEN** the command fails with an error naming the file, and the file's bytes are unchanged

### Requirement: A missing state file starts an empty environment
When an environment has no `state.json`, the system SHALL start it with no variables and SHALL NOT report an error.

#### Scenario: No state file
- **WHEN** the environment's `state.json` does not exist and a command is run
- **THEN** the command runs with no variables
