## ADDED Requirements

### Requirement: Optimizer state lives where the parameter lives
A parameter on a device SHALL keep its Adam moments on the same device, allocated on first use and released by `Tape.Close`. A step SHALL not read the state back.

#### Scenario: State stays resident
- **WHEN** several Adam steps run on a device parameter
- **THEN** no moment is read back to the host between steps
