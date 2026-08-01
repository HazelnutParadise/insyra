## ADDED Requirements

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
