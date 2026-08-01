## ADDED Requirements

### Requirement: Dataset fingerprint cost is proportional to column bytes
The system SHALL compute a dataset fingerprint in time proportional to the size of the column data, and SHALL NOT render column values to text in order to fingerprint them.

#### Scenario: A large numeric column is projected
- **WHEN** a numeric column is projected into a dataset
- **THEN** the fingerprint is derived from the binary representation of the values
- **AND** computing it does not allocate a decimal rendering of the column

#### Scenario: Fingerprint cost is measurable
- **WHEN** the repository is tested
- **THEN** a benchmark reports the cost of fingerprinting a large numeric column, so the figure is checkable rather than asserted

### Requirement: Fingerprints remain content-addressed over every value
The system SHALL include every value of every column in the fingerprint, and SHALL NOT substitute sampling, length-only, or identity-based shortcuts.

#### Scenario: Two datasets differ in a single value
- **WHEN** two datasets are identical except for one value in one column
- **THEN** their fingerprints differ

#### Scenario: The same data is projected twice
- **WHEN** the same column contents are projected twice within a session
- **THEN** both projections produce the same fingerprint
- **AND** the cache reports one resident entry rather than two

#### Scenario: Two string columns share their concatenated bytes
- **WHEN** two string columns hold the same bytes divided between values differently, such as `["ab", "c"]` and `["a", "bc"]`
- **THEN** their fingerprints differ
