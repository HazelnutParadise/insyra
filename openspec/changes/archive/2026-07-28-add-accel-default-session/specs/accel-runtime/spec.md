## ADDED Requirements

### Requirement: A process-shared session is available without construction
The system SHALL provide a session shared by the process, created on first use, that any caller can obtain without owning its lifetime.

#### Scenario: Two callers ask for the shared session
- **WHEN** two callers request the process-shared session
- **THEN** both receive the same session
- **AND** device discovery has run once, not once per caller

#### Scenario: Importing the package
- **WHEN** a program imports the accel package but never requests the shared session
- **THEN** no device is opened and no discovery runs

#### Scenario: No accelerator is present
- **WHEN** the shared session is requested on a host with no usable device
- **THEN** a usable session is still returned
- **AND** its report explains why acceleration is unavailable

#### Scenario: A caller closes the shared session
- **WHEN** a caller calls Close on the process-shared session
- **THEN** the call succeeds without error
- **AND** the session remains usable for every other caller
