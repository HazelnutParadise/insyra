## ADDED Requirements

### Requirement: A model can declare that its predictions are group assignments
The system SHALL let a model state that what it predicts is a grouping rather than a measurement or a class from a known set.

#### Scenario: A clustering model is inspected
- **WHEN** a caller holds a fitted clustering model
- **THEN** it can determine that the model assigns groups rather than predicting values
- **AND** how many groups it converged on

#### Scenario: A regression metric is applied to a clustering model
- **WHEN** a regression metric is used to score a model that declares itself a clusterer
- **THEN** the request is refused with an error naming the mismatch
- **AND** no score is produced

#### Scenario: A model that declares nothing
- **WHEN** a model implements neither the classifier nor the clusterer declaration
- **THEN** it is scored as before, with no change in behaviour
