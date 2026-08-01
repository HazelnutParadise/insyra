## ADDED Requirements

### Requirement: Sample weights flow through cross-validation aligned with their rows
The system SHALL cross-validate weighted estimators by subsetting the caller's weights with each fold's training rows, so every training row is fitted under its own weight.

#### Scenario: A fold is fitted

- **WHEN** a weighted cross-validation fits any fold
- **THEN** the estimator receives that fold's training rows and exactly those rows' weights, in the same order

#### Scenario: The held-out rows are scored

- **WHEN** a fold's held-out rows are scored
- **THEN** the metric is computed unweighted
- **AND** the documentation states this, matching the reference implementation's default

#### Scenario: The estimator does not accept weights

- **WHEN** the supplied estimator has no weighted fitting function
- **THEN** the request is refused rather than silently fitted unweighted

#### Scenario: The weights are invalid

- **WHEN** any weight is missing, zero, negative or not finite, or the count does not match the rows
- **THEN** the request is refused with an error locating the problem

### Requirement: The weights channel is optional and breaks nothing
The system SHALL keep every existing estimator, pipeline and grid search valid: an estimator declares weight support by setting the optional function, and everything that does not mention weights behaves as before.

#### Scenario: An existing estimator is used unweighted

- **WHEN** an estimator without the weighted fitting function runs through unweighted cross-validation
- **THEN** its behaviour is unchanged
