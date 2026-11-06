## ADDED Requirements

### Requirement: A new environment's files are never seen part-written
Create SHALL make each default file appear whole, so a session opening the environment while another creates it reads a complete `state.json`.

#### Scenario: Two sessions open a missing environment
- **WHEN** two sessions open the same missing environment at once
- **THEN** both open it, and neither reports that `state.json` cannot be read
