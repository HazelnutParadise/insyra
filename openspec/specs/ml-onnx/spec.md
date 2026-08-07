# ml-onnx Specification

## Purpose
TBD - created by archiving change add-ml-onnx-export. Update Purpose after archive.
## Requirements
### Requirement: A fitted model can be exported for other runtimes
The system SHALL write a fitted model in a form other machine-learning runtimes read.

#### Scenario: A linear or logistic model is exported
- **WHEN** a caller exports a fitted linear or logistic model
- **THEN** the written model scores identically to this package on the same observations, within the tolerance of the exchange format's own precision
- **AND** it loads in a standard runtime without modification

#### Scenario: A tree model is exported
- **WHEN** a caller exports a fitted tree
- **THEN** the written model reproduces this package's predictions on the same observations
- **AND** its missing-value routing is preserved

#### Scenario: A pipeline is exported
- **WHEN** a caller exports a fitted pipeline
- **THEN** the preprocessing and the model are written as one graph
- **AND** scoring the graph on raw observations reproduces what the pipeline produces

#### Scenario: A model has no equivalent in the exchange format
- **WHEN** a caller exports a model the format cannot express
- **THEN** the export is refused with an error naming the model
- **AND** nothing is written

### Requirement: The export is verified by round trip
The system SHALL check its exports against an independent runtime rather than against its own reading of them.

#### Scenario: An exported model is checked
- **WHEN** a model is exported
- **THEN** an independent runtime scores the written file
- **AND** its predictions are compared against this package's on the same observations

### Requirement: Penalized and weighted linear fits export
The system SHALL export ridge, lasso and weighted linear models as ONNX linear regressors that an independent runtime evaluates to the model's own predictions.

#### Scenario: A penalized linear model is exported

- **WHEN** a fitted ridge, lasso or weighted linear model is exported and run by an independent runtime
- **THEN** the runtime's predictions match the model's within the exchange format's precision

### Requirement: Ensembles export as tree ensembles
The system SHALL export random forests and gradient-boosted ensembles as ONNX tree ensembles whose aggregation reproduces the model's own: averaged probabilities for a forest classifier, averaged values for a forest regressor, base value plus learning-rate-scaled leaf sums for boosting.

#### Scenario: A forest is exported

- **WHEN** a fitted forest is exported and run by an independent runtime
- **THEN** regression predictions match the tree average and classification labels match the averaged-probability argmax

#### Scenario: A boosted regressor is exported

- **WHEN** a fitted boosted regressor is exported and run
- **THEN** the runtime's sum over scaled leaves plus the base value matches the model's prediction

#### Scenario: A boosted binary classifier is exported

- **WHEN** a fitted boosted binary classifier is exported and run
- **THEN** the runtime's probabilities match the model's sigmoid of the log-odds and the label matches the half threshold

### Requirement: Every exported family is proved by execution
The system SHALL include every exportable family in the independent-runtime round trip, so an export that no runtime accepts cannot be archived as verified.

#### Scenario: The round trip runs

- **WHEN** the round trip runs where the runtime is installed
- **THEN** every exportable family is executed, numeric outputs compared within single-precision tolerance and labels exactly

