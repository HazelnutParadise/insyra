# ml-pipeline Specification

## Purpose
TBD - created by archiving change add-ml-pipeline. Update Purpose after archive.
## Requirements
### Requirement: Preprocessing and a model fit and predict as one
The system SHALL let a caller compose preprocessing steps with a model into a single fitted object.

#### Scenario: A pipeline is fitted and then scores new data
- **WHEN** a caller fits a pipeline of preprocessing steps and a model on training data
- **AND** scores new observations through the fitted pipeline
- **THEN** each step's fitted parameters are applied to the new observations before the model sees them
- **AND** the result equals applying each step and the model separately in the same order

#### Scenario: A fitted pipeline is used where a model is expected
- **WHEN** a fitted pipeline is passed to something that accepts a model
- **THEN** it is accepted, and reports the columns the first step was fitted on

#### Scenario: A step fails during fitting
- **WHEN** fitting a step returns an error
- **THEN** the pipeline reports which step failed, by name

#### Scenario: The preprocessing is refitted
- **WHEN** the same pipeline definition is fitted twice on different data
- **THEN** the second fit derives its parameters from the second data only

### Requirement: Different columns can be treated differently
The system SHALL let a transformer apply to a named subset of columns while the rest pass through.

#### Scenario: Numeric and categorical columns are treated differently in one pipeline
- **WHEN** a caller scales the numeric columns and encodes the categorical ones in the same pipeline
- **THEN** each transformer sees only the columns it was given
- **AND** the columns it was not given reach the model unchanged

#### Scenario: A named column is absent
- **WHEN** a transformer is scoped to a column the data does not contain
- **THEN** the request is refused with an error naming the column

### Requirement: A pipeline reports the columns its estimator was fitted on
The system SHALL make readable, from a fitted pipeline, the names of the columns the final estimator saw after every step had been applied.

#### Scenario: A step changes the column count

- **WHEN** a pipeline whose preprocessing expands one column into several is fitted
- **THEN** the columns the final estimator saw are readable from the fitted pipeline, in the order it saw them
- **AND** they are distinct from the columns the pipeline itself was fitted on

#### Scenario: A pipeline has no preprocessing steps

- **WHEN** a pipeline with no steps is fitted
- **THEN** the columns its estimator saw are the columns the pipeline was fitted on

#### Scenario: Importances are read from a pipeline

- **WHEN** a fitted pipeline reports feature importances
- **THEN** there is exactly one importance for each column the estimator saw
- **AND** the two are in the same order

### Requirement: A model's importances match its feature names in number
The system SHALL require that any model reporting feature importances reports as many as it has feature names, and SHALL check this of a model presented for conformance.

#### Scenario: A model reports more importances than it has features

- **WHEN** a model under conformance check reports a different number of importances than feature names
- **THEN** the check fails and names both counts

### Requirement: A pipeline in cross-validation is fitted on training rows only
The system SHALL fit every preprocessing step of a pipeline on each fold's training rows alone, never on the held-out rows.

#### Scenario: A pipeline is cross-validated

- **WHEN** a pipeline containing a fitted-parameter preprocessing step is cross-validated
- **THEN** each fold's step is fitted from that fold's training rows only
- **AND** a held-out row's transformed value is what the training-fitted step produces for it, not what a step fitted on the whole dataset would produce

