## ADDED Requirements

### Requirement: A check that could not run says so in a way that fails
The system SHALL provide a mode in which any verification skipped for a missing reference implementation fails the run instead, naming the missing tool and the verification that went unperformed.

#### Scenario: A reference toolchain is absent under strict mode

- **WHEN** the suite runs with reference-toolchain verification required and the reference implementation is absent
- **THEN** the affected check fails rather than skipping
- **AND** the failure names both the missing tool and what was left unverified

#### Scenario: A reference toolchain is absent by default

- **WHEN** the suite runs without reference-toolchain verification required and the reference implementation is absent
- **THEN** the affected check skips as before
- **AND** the run still passes, so the suite remains usable on a machine that has none of them

#### Scenario: A check is behind an opt-in flag

- **WHEN** a check is opt-in because its reference implementation is usually absent, and the suite runs with reference-toolchain verification required
- **THEN** that check runs rather than skipping, because the reason it was opt-in no longer holds

### Requirement: Every reference-toolchain gate is subject to the mode
The system SHALL route every check that depends on an external reference implementation through the same gate, so no such check can skip silently.

#### Scenario: A verification depends on R

- **WHEN** a check requires R with its analysis packages
- **THEN** its absence is reported through the shared gate

#### Scenario: A verification depends on Python

- **WHEN** a check requires Python with a scientific stack, scikit-learn, or an ONNX runtime
- **THEN** its absence is reported through the shared gate

### Requirement: Continuous integration provides the toolchains it gates on
The system SHALL install, in every workflow that exists to run a reference-implementation comparison, every dependency that comparison's own gate requires.

#### Scenario: A parity workflow runs its suite

- **WHEN** a workflow exists to run a cross-language parity suite
- **THEN** the dependencies it installs satisfy that suite's gate
- **AND** the suite executes rather than skipping

#### Scenario: The full verification set runs in continuous integration

- **WHEN** continuous integration runs the reference-implementation verifications
- **THEN** it does so with reference-toolchain verification required
- **AND** a check that could not run fails the workflow
