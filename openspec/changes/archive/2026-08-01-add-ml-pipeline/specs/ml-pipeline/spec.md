## ADDED Requirements

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
