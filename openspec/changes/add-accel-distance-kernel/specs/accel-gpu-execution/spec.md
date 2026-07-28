## ADDED Requirements

### Requirement: Squared distance from rows to query points
The system SHALL compute the squared Euclidean distance from every row of a dataset to each of a supplied set of query points.

#### Scenario: Distances are requested for a dataset and query points
- **WHEN** a caller requests squared distances for a dataset of numeric columns and one or more query points of matching dimension
- **THEN** the runtime returns one distance per query point per row
- **AND** each distance equals the sum over dimensions of the squared difference between the row and the query point

#### Scenario: A query point has the wrong dimension
- **WHEN** a query point's length does not match the number of columns in the dataset
- **THEN** the runtime refuses the request rather than computing against missing dimensions

#### Scenario: No query points are supplied
- **WHEN** distances are requested with no query points
- **THEN** the runtime refuses the request

### Requirement: A device result is bit-identical to its CPU reference
The system SHALL ship a CPU reference implementation for every device operation, and SHALL verify on the running platform that the device result matches it bit for bit.

#### Scenario: The kernel runs on a host with a device
- **WHEN** the squared-distance kernel executes on a device
- **THEN** every returned value is bit-identical to the CPU reference computed over the same inputs

#### Scenario: A platform cannot reach parity
- **WHEN** a platform's device result differs from the CPU reference
- **THEN** the difference is observable rather than silent
