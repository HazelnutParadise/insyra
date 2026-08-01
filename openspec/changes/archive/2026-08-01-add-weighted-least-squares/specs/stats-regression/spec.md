## ADDED Requirements

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
