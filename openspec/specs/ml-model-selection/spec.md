# ml-model-selection Specification

## Purpose
TBD - created by archiving change add-ml-model-selection. Update Purpose after archive.
## Requirements
### Requirement: Data can be split for honest evaluation
The system SHALL divide observations into folds for evaluation.

#### Scenario: Observations are divided into k folds
- **WHEN** a caller splits observations into k folds
- **THEN** every observation appears in exactly one fold
- **AND** no observation is lost or duplicated

#### Scenario: A split is repeated with the same seed
- **WHEN** the same data is split twice with the same seed
- **THEN** the folds are identical

#### Scenario: Class balance is preserved
- **WHEN** a caller asks for a stratified split of labelled observations
- **THEN** each fold carries approximately the class proportions of the whole

#### Scenario: A class is too small to stratify
- **WHEN** a class has fewer members than the number of folds
- **THEN** the request is refused with an error naming the class

### Requirement: A model is evaluated across folds
The system SHALL evaluate an estimator by fitting it on each training fold and scoring it on the held-out one.

#### Scenario: An estimator is cross-validated
- **WHEN** a caller cross-validates an estimator over k folds
- **THEN** the estimator is fitted k times, each on the data excluding one fold
- **AND** a score is returned for each fold

#### Scenario: A pipeline is cross-validated
- **WHEN** the estimator is a pipeline containing preprocessing
- **THEN** the preprocessing is refitted on each training fold
- **AND** no fold's preprocessing derives from data outside its training fold

#### Scenario: A fold fails to fit
- **WHEN** fitting fails on one fold
- **THEN** the failure is reported with the fold it occurred on rather than silently reducing the fold count

### Requirement: A reported score names what it measured
The system SHALL require the metric to be stated rather than inferring it from the kind of model.

#### Scenario: A model is scored
- **WHEN** a caller scores a model
- **THEN** the metric is supplied by the caller
- **AND** the result carries the name of the metric that produced it

#### Scenario: A metric does not apply to a model
- **WHEN** a classification metric is applied to a model that produces continuous values
- **THEN** the request is refused with an error rather than producing a meaningless number

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

