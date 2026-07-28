## ADDED Requirements

### Requirement: Exact nearest query points
The system SHALL report the nearest query points per row with the same result a pure `float64` computation would produce, whether or not a device took part.

#### Scenario: The nearest query points are requested
- **WHEN** a caller asks for the `m` nearest query points for every row
- **THEN** the runtime returns `m` query indices and `m` `float64` squared distances per row, nearest first
- **AND** the indices and distances equal those of a `float64` computation over every query point

#### Scenario: A device narrows the candidates
- **WHEN** a device is available and the workload is eligible
- **THEN** the device ranks the query points in single precision and returns a shortlist per row
- **AND** the returned distances are recomputed in `float64` before any of them is chosen

#### Scenario: The shortlist boundary cannot be trusted
- **WHEN** the distance of the best rejected candidate lies within the single-precision error bound of the last accepted one
- **THEN** that row is recomputed against every query point in `float64`
- **AND** the error bound grows with the number of dimensions rather than being a fixed constant

#### Scenario: Two query points are equally close
- **WHEN** a row is exactly equidistant from more than one query point in `float64`
- **THEN** the lowest of those query indices is reported first

#### Scenario: How much rechecking happened is reported
- **WHEN** the operation completes
- **THEN** the number of rows recomputed against every query point is reported

#### Scenario: More neighbours are requested than exist
- **WHEN** a caller asks for more nearest query points than there are query points
- **THEN** the request is rejected before any work is scheduled
