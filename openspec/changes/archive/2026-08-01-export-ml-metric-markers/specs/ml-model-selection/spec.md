## ADDED Requirements

### Requirement: A metric defined outside the package can state what it needs
The system SHALL let any metric, wherever it is defined, declare whether it requires class labels or probabilities.

#### Scenario: A metric defined outside the package requires probabilities
- **WHEN** a metric implemented outside the package declares that it requires probabilities
- **AND** it is used to evaluate a model that can produce them
- **THEN** it receives the probabilities

#### Scenario: A metric requires probabilities and the model cannot produce them
- **WHEN** a metric requiring probabilities is used with a model that does not produce them
- **THEN** the request is refused with an error rather than passing values of a different kind

#### Scenario: A metric declares nothing
- **WHEN** a metric declares neither requirement
- **THEN** it receives the model's predictions, as before

#### Scenario: A caller inspects what a prediction carries
- **WHEN** a metric receives a prediction
- **THEN** which of its fields are populated is determined by what the metric declared and what the model can supply
- **AND** that relationship is documented rather than left to be inferred from a nil
