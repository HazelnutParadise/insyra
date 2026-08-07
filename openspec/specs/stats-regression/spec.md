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

### Requirement: Coefficients can be penalized
The system SHALL fit linear models under an L2 penalty (ridge) and an L1 penalty (lasso), with the penalty strength chosen by the caller, the intercept never penalized, and no feature standardisation applied.

#### Scenario: Predictors are collinear

- **WHEN** a ridge fit is requested on predictors whose unpenalized normal equations are singular
- **THEN** coefficients are returned for any positive penalty
- **AND** the same data through the unpenalized estimator is refused

#### Scenario: The penalty is zero

- **WHEN** a ridge fit is requested with a zero penalty
- **THEN** its coefficients equal the unpenalized least-squares coefficients

#### Scenario: The penalty drives coefficients to zero

- **WHEN** a lasso fit is requested with a penalty large enough that a predictor does not earn its coefficient
- **THEN** that coefficient is exactly zero, not merely small

#### Scenario: An invalid penalty is supplied

- **WHEN** the penalty is negative, or not a finite number
- **THEN** the fit is refused with an error

#### Scenario: The iterative fit does not converge

- **WHEN** a lasso fit reaches its iteration cap before its tolerance
- **THEN** the result still returns, reporting that it did not converge and how many iterations ran

#### Scenario: A caller looks for standard errors on a penalized fit

- **THEN** the penalized result types carry no standard errors, t values or p values
- **AND** the documentation states that classical inference does not apply to penalized estimates

### Requirement: Penalized fits match the scikit-learn reference
The system SHALL produce, for the same data and penalty, the same coefficients, intercept and predictions as scikit-learn's Ridge and Lasso, verified where the reference is installed, and SHALL record that scikit-learn rather than glmnet is the reference because the two scale their penalties differently.

#### Scenario: The reference is present

- **WHEN** the comparison runs on a machine with scikit-learn available
- **THEN** ridge coefficients match to numerical precision and lasso coefficients match within the shared convergence tolerance

#### Scenario: The reference is absent

- **WHEN** scikit-learn is not available
- **THEN** the comparison reports through the shared reference-toolchain gate

### Requirement: Observations can be weighted in a linear fit
The system SHALL fit linear models where each observation carries a caller-supplied positive weight, with classical inference computed under those weights.

#### Scenario: Uniform weights are supplied

- **WHEN** every observation carries the same weight
- **THEN** the coefficients equal the unweighted least-squares coefficients

#### Scenario: Weights differ

- **WHEN** observations carry different weights
- **THEN** the fit minimises the weighted sum of squared residuals
- **AND** standard errors, t and p values are computed under those weights

#### Scenario: An invalid weight is supplied

- **WHEN** any weight is zero, negative, or not a finite number
- **THEN** the fit is refused with an error naming the row
- **AND** no exclusion semantics are guessed for a zero weight

#### Scenario: The weighted fit is compared to the reference implementation

- **WHEN** the comparison runs where the Python reference is installed
- **THEN** coefficients, standard errors, t and p values, weighted R² and predictions match statsmodels' WLS

