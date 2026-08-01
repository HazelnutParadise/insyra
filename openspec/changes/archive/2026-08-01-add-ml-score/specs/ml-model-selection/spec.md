## ADDED Requirements

### Requirement: A fitted model is scored on held-out data without refitting
The system SHALL evaluate an already-fitted model against observations and their true values using a supplied metric, without fitting anything.

#### Scenario: A caller scores a model they already hold

- **WHEN** a caller supplies a fitted model, held-out observations, the true values for those observations, and a metric
- **THEN** the metric's score for that model on that data is returned
- **AND** no fitting occurs

#### Scenario: The metric needs something the model does not provide

- **WHEN** the supplied metric requires class probabilities and the model does not report them
- **THEN** the request is refused with an error before any prediction is made
- **AND** the refusal is the same one cross-validation makes for the same pairing

#### Scenario: The metric needs class labels from a model that reports probabilities

- **WHEN** the supplied metric scores class labels and the model reports probabilities rather than labels
- **THEN** the labels are derived on the metric's behalf
- **AND** the derivation is identical to the one cross-validation performs

#### Scenario: Observations and true values disagree in length

- **WHEN** the number of observations does not match the number of true values
- **THEN** the request is refused with an error naming both counts
