## ADDED Requirements

### Requirement: Penalized regression is fitted through the protocol
The system SHALL expose ridge and lasso fitting through the same fitting-function shape as the other regression families, returning models that score new observations by feature name.

#### Scenario: A penalized model is fitted and scores new data

- **WHEN** a caller fits ridge or lasso through the package on a feature table
- **THEN** the returned model predicts through the same method as every other model
- **AND** it passes the protocol conformance checks
