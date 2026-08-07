## ADDED Requirements

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
