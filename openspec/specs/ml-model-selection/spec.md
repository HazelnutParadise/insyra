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

### Requirement: A metric declares which direction is better
The system SHALL require every metric to declare whether a larger score is better, a smaller score is better, or the metric has no scalar direction. The system SHALL NOT infer the direction from the metric's name or kind.

#### Scenario: A built-in metric is asked its direction

- **WHEN** a caller asks any metric supplied by this package which direction is better
- **THEN** it answers: larger for accuracy, R² and area under the ROC curve; smaller for root mean squared error, mean absolute error and logarithmic loss
- **AND** the confusion matrix answers that it has no direction

#### Scenario: A metric from outside the package is used

- **WHEN** a caller supplies a metric they defined themselves
- **THEN** it must declare a direction to be accepted
- **AND** the declaration is not guessed from its name

#### Scenario: A metric returns a scalar score but declares no direction

- **WHEN** a metric declares no direction yet returns a score that is a number
- **THEN** the request is refused with an error rather than a direction being assumed

### Requirement: Two results are compared by the metric's own direction
The system SHALL provide a comparison that reports which of two scores is better, using the direction the metric declared.

#### Scenario: A caller selects between two models scored by a loss metric

- **WHEN** two cross-validation results for the same loss metric are compared
- **THEN** the one with the smaller mean is reported as better

#### Scenario: A caller selects between two models scored by a gain metric

- **WHEN** two cross-validation results for the same gain metric are compared
- **THEN** the one with the larger mean is reported as better

#### Scenario: Results from different metrics are compared

- **WHEN** two results produced by different metrics are compared
- **THEN** the comparison is refused with an error naming both metrics

#### Scenario: A directionless metric's results are compared

- **WHEN** two results from a metric that declares no direction are compared
- **THEN** the comparison is refused with an error

### Requirement: A cross-validation result carries its metric's direction
The system SHALL record the direction on the result, so a result read apart from the metric that produced it remains interpretable.

#### Scenario: A result is read without the metric value

- **WHEN** a caller holds a cross-validation result and no longer holds the metric
- **THEN** the direction is readable from the result

### Requirement: Per-class classification quality is measurable
The system SHALL score class predictions with precision, recall and F1, combined across classes by a caller-chosen averaging mode, and SHALL declare that a larger score is better for all three.

#### Scenario: No averaging mode is chosen

- **WHEN** a caller scores predictions without choosing an averaging mode
- **THEN** the unweighted mean over every observed class is returned
- **AND** this works for any number of classes without naming one

#### Scenario: Micro and weighted averaging are requested

- **WHEN** a caller chooses micro or support-weighted averaging
- **THEN** micro combines the per-class counts before dividing
- **AND** weighted combines the per-class scores in proportion to how often each class actually occurs

#### Scenario: One class is scored

- **WHEN** a caller chooses binary averaging and names the positive class
- **THEN** precision, recall and F1 are computed for that class alone

#### Scenario: Binary averaging without a named positive class

- **WHEN** a caller chooses binary averaging without naming the positive class
- **THEN** the request is refused with an error
- **AND** no class is chosen on the caller's behalf, because the score is not invariant under that choice

#### Scenario: A positive class is named under a non-binary average

- **WHEN** a positive class is named but the averaging mode does not use one
- **THEN** the request is refused rather than the name being silently ignored

#### Scenario: A class is never predicted

- **WHEN** some class occurs in the true labels but never in the predictions
- **THEN** its precision contributes zero rather than failing the whole evaluation
- **AND** the convention is documented

### Requirement: The per-class metrics agree with the reference implementation
The system SHALL produce the same precision, recall and F1 as scikit-learn's reference computation for every averaging mode, verified where the reference is installed.

#### Scenario: The reference toolchain is present

- **WHEN** the comparison runs on a machine with scikit-learn available
- **THEN** macro, micro, weighted and binary results match the reference

#### Scenario: The reference toolchain is absent

- **WHEN** scikit-learn is not available
- **THEN** the comparison reports through the shared reference-toolchain gate rather than skipping on its own

