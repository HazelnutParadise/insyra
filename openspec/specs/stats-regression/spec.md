# stats-regression Specification

## Purpose
TBD - created by archiving change add-stats-regression-predict. Update Purpose after archive.
## Requirements
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

### Requirement: A fitted generalised linear model publishes its link
The system SHALL let a caller read which link function a fitted model was estimated under.

#### Scenario: A logistic or Poisson model is inspected
- **WHEN** a caller holds a fitted logistic or Poisson regression result
- **THEN** the link function it was fitted with is readable from the result
- **AND** it is named and typed the same way the general GLM result names it

#### Scenario: The link is used to reproduce a prediction outside the package
- **WHEN** code outside `stats` applies the published link to the linear predictor
- **THEN** it obtains the same response the result's own prediction returns

### Requirement: Numeric input that cannot be read is refused, never replaced
The system SHALL refuse any observation it cannot convert to a finite number, and SHALL NOT substitute a value of its own choosing.

This applies to every numeric input a statistical routine reads: predictors, targets, offsets, and the paired series of a correlation.

#### Scenario: A predictor column contains a value that is not a number

- **WHEN** a regression is fitted on data where a predictor holds a missing value, a blank, or text
- **THEN** the request is refused with an error
- **AND** the error names the column and the row where the value was found
- **AND** no coefficient is returned

#### Scenario: A target column contains a value that is not a number

- **WHEN** a regression is fitted on data where the target holds a missing value, a blank, or text
- **THEN** the request is refused with an error naming the row
- **AND** the value is not treated as zero

#### Scenario: A correlation is computed over data containing a blank

- **WHEN** a correlation coefficient is requested for two series and either contains a value that is not a finite number
- **THEN** the request is refused with an error
- **AND** no coefficient is returned

#### Scenario: An infinite or undefined value reaches a numeric input

- **WHEN** an input holds an infinity or an undefined numeric value
- **THEN** it is refused on the same terms as a value that is not a number

### Requirement: Each family states how it treats values it cannot read
The system SHALL document, for every statistical family it exposes, which of its three treatments applies to unreadable values: refusal, removal of the whole observation, or a learned direction.

#### Scenario: A caller asks what happens to their missing values

- **WHEN** a caller consults the documentation for a statistical family
- **THEN** that family's treatment of unreadable values is stated there
- **AND** where a family removes observations rather than refusing them, the documentation says so

