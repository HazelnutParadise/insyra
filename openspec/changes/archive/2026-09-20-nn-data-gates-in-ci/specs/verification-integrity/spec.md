## ADDED Requirements

### Requirement: Continuous integration supplies the data its gates need
The system SHALL run, for every check gated on a dataset or model file the repository does not carry, a workflow that fetches that data from a pinned source, verifies it against a recorded checksum before any test runs, and executes the check. Such a workflow SHALL fail when a gated check skipped instead of running. A check gated on hardware no hosted runner has SHALL stay manual and SHALL be documented as manual rather than left to look covered.

#### Scenario: A gate needs files too large to commit

- **WHEN** a check compares against published checkpoints or a dataset that the checkout does not contain
- **THEN** the workflow fetches each file from a URL naming an immutable commit, revision, or object
- **AND** verifies every file against its recorded sha256 before running the check
- **AND** a file whose checksum does not match fails the workflow before any test runs

#### Scenario: The data is present but the gate skipped anyway

- **WHEN** a gated test skips in that workflow, for instance because the variable naming its data directory is wrong
- **THEN** the workflow fails and names the test that did not run

#### Scenario: A gate needs hardware no runner has

- **WHEN** a check requires a GPU
- **THEN** no workflow claims to run it
- **AND** the documentation records how to run it by hand
