## MODIFIED Requirements

### Requirement: Continuous integration provides the toolchains it gates on
The system SHALL install, in every workflow that exists to run a reference-implementation comparison, every dependency that comparison's own gate requires. Every opt-in comparison that strict mode turns on SHALL be run by a step of such a workflow.

#### Scenario: A parity workflow runs its suite

- **WHEN** a workflow exists to run a cross-language parity suite
- **THEN** the dependencies it installs satisfy that suite's gate
- **AND** the suite executes rather than skipping

#### Scenario: The full verification set runs in continuous integration

- **WHEN** continuous integration runs the reference-implementation verifications
- **THEN** it does so with reference-toolchain verification required
- **AND** a check that could not run fails the workflow

#### Scenario: An opt-in comparison has a step that runs it

- **WHEN** a test is opt-in because its reference implementation is usually absent, such as the portfolio comparison against cvxpy
- **THEN** the reference verification workflow installs that implementation and has a step whose `go test` pattern selects the test
- **AND** the test executes in that step rather than skipping
