## ADDED Requirements

### Requirement: Nearest query point per row
The system SHALL report, for every row, which of the supplied query points is closest and how far away it is.

#### Scenario: Nearest query points are requested
- **WHEN** a caller requests the nearest query point for a dataset and one or more query points
- **THEN** the runtime returns one query index and one squared distance per row
- **AND** the reported distance is the smallest squared Euclidean distance from that row to any query point

#### Scenario: Two query points are equally close
- **WHEN** a row is exactly equidistant from more than one query point
- **THEN** the lowest of those query indices is reported

#### Scenario: The result is reduced on the device
- **WHEN** the operation runs on a device
- **THEN** the amount of data read back grows with the number of rows and not with the number of query points
