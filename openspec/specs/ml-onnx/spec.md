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

