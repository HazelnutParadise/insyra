## ADDED Requirements

### Requirement: The reference toolchains are set up in one place

Every workflow that runs a cross-language comparison SHALL set up Python, R and their shared packages through the one composite action, and SHALL add only the packages it needs beyond that set.

#### Scenario: A package the gates need is added
- **WHEN** a Python or R package is added to the shared set
- **THEN** the change is one edit, and every comparison workflow installs it

#### Scenario: A workflow needs more than the shared set
- **WHEN** a workflow needs packages the others do not
- **THEN** it passes them to the action rather than repeating the setup
