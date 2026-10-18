## ADDED Requirements

### Requirement: Concurrent saves keep each other's variables
When a process saves an environment's variables after restoring them, the system SHALL write the variables that process added or changed and remove the ones it removed, leaving the variables another process saved in between unchanged.

#### Scenario: Two commands add different variables
- **WHEN** two processes restore the same environment, one adds `from_a` and the other adds `from_b`, and both save
- **THEN** the saved state holds both `from_a` and `from_b`

#### Scenario: One process removes a variable
- **WHEN** a process removes a variable it restored and saves
- **THEN** the variable is absent from the saved state, and variables it did not touch are kept

### Requirement: Saves do not collide on a temporary file
The system SHALL write each save through a temporary file named for that writer and SHALL leave no temporary or lock file behind.

#### Scenario: Twelve simultaneous saves
- **WHEN** twelve processes each restore, add one variable and save
- **THEN** every save succeeds and the saved state holds all twelve variables
