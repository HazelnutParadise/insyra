## ADDED Requirements

### Requirement: A fitted decomposition can be applied to new data
The system SHALL return enough of a fitted principal-component decomposition to project observations it was not fitted on.

#### Scenario: New observations are projected
- **WHEN** a caller applies a fitted decomposition's centring, scaling and loadings to new observations
- **THEN** the resulting scores match what R's `predict` returns for the same model and data

#### Scenario: The decomposition was fitted on correlations rather than covariances
- **WHEN** a fit standardised its columns
- **THEN** the scaling it applied is returned alongside the centring
- **AND** a caller can tell the two kinds of fit apart from the result alone

#### Scenario: The training scores are wanted
- **WHEN** a caller wants the transformed training data
- **THEN** the scores computed during the fit are available without recomputing them
