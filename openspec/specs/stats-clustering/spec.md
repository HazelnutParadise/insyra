# stats-clustering Specification

## Purpose
TBD - created by archiving change add-kmeans-assign. Update Purpose after archive.
## Requirements
### Requirement: A fitted clustering assigns new observations
The system SHALL let a caller assign observations to the centres a clustering converged on, whether or not those observations were part of the fit.

#### Scenario: New observations are assigned
- **WHEN** a caller supplies observations to a fitted KMeans result
- **THEN** the index of the nearest centre is returned for each
- **AND** the squared distance to it is returned alongside

#### Scenario: The training data is assigned back
- **WHEN** a caller assigns the same observations the model was fitted on
- **THEN** the result agrees with the cluster assignment the fit reported

#### Scenario: An observation is equidistant from two centres
- **WHEN** an observation is exactly equidistant from more than one centre
- **THEN** the lowest of those centre indices is reported

#### Scenario: The observation shape does not match the fit
- **WHEN** observations carry a different number of columns than the fitted centres
- **THEN** the request is refused with an error naming the mismatch

#### Scenario: The answer does not depend on where it was computed
- **WHEN** the assignment is computed
- **THEN** the result is what a `float64` computation over every centre would produce

