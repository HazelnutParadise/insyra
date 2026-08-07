## ADDED Requirements

### Requirement: Every fitted regression can score new data
The system SHALL let a caller obtain predictions from any fitted regression result.

#### Scenario: A linear or polynomial model scores new observations
- **WHEN** a caller supplies new predictor values to a fitted linear or polynomial regression
- **THEN** the predicted response is returned for each observation
- **AND** the values equal what R's `predict()` returns for the same model and data

#### Scenario: A model fitted on transformed data scores new observations
- **WHEN** a caller supplies new predictor values to a fitted exponential or logarithmic regression
- **THEN** the prediction is returned on the scale of the original response, not the transformed one
- **AND** the values equal what R returns for the equivalent model

#### Scenario: The predictor count does not match the fitted model
- **WHEN** a caller supplies a different number of predictors than the model was fitted with
- **THEN** the request is refused with an error naming the mismatch

### Requirement: The prediction path is validated against a reference implementation
The system SHALL check its predictions against R, in the same way its fitting is already checked.

#### Scenario: Any regression family gains a prediction
- **WHEN** a regression family offers a prediction
- **THEN** a reference script produces R's predictions for the same model and data
- **AND** the test compares Insyra's predictions against them field by field

#### Scenario: The reference returns more than a point estimate
- **WHEN** R's `predict()` returns standard errors or intervals that Insyra does not produce
- **THEN** the difference is recorded as a known gap rather than left undocumented
