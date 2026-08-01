## ADDED Requirements

### Requirement: Boosted tree ensembles can be fitted
The system SHALL fit gradient-boosted tree ensembles: regression under squared loss by residual fitting, and binary classification under logistic loss with second-order leaf values.

#### Scenario: More stages fit the training data no worse

- **WHEN** two boosted regressors differ only in stage count
- **THEN** the one with more stages fits the training data at least as well

#### Scenario: A boosted classifier reports probabilities

- **WHEN** a boosted binary classifier reports probabilities
- **THEN** each row's two probabilities sum to one
- **AND** the predicted label is the class whose probability exceeds one half

#### Scenario: The residuals run out early

- **WHEN** the residuals reach zero before the requested stage count
- **THEN** fitting stops and the model reports how many stages ran

#### Scenario: A multiclass target is supplied

- **WHEN** the classification target holds more than two classes
- **THEN** the fit is refused with an error naming the binary limit

### Requirement: Boosting is deterministic
The system SHALL produce identical boosted ensembles from identical data and options, with no randomness involved.

#### Scenario: The same fit runs twice

- **WHEN** the same data and options are fitted twice
- **THEN** the predictions are identical, with no seed to manage
