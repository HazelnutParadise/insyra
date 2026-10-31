## ADDED Requirements

### Requirement: The hazard is fitted from person-period data
`stats.DiscreteTimeHazard` SHALL expand each spell of d whole periods into d rows, mark the last row 1 only when the spell ended, and fit a binomial GLM of that indicator on the chosen baseline in the period and the covariates, with a logit or complementary log-log link.

#### Scenario: A censored spell
- **WHEN** a spell lasts 3 periods and is censored
- **THEN** it contributes three rows with y equal to 0

### Requirement: Results agree with R
`stats.DiscreteTimeHazard` SHALL give the coefficients, standard errors and deviance of R's `glm` with the binomial family and the same link on the same person-period data.

#### Scenario: Logit with a linear baseline
- **WHEN** the model has a term in t and one covariate
- **THEN** every coefficient and standard error equals R's within 1e-6

### Requirement: Unusable input is refused
`stats.DiscreteTimeHazard` SHALL return an error naming the row index for a duration that is not a whole number of at least 1, for an observed flag that is neither 0, 1 nor a boolean, and for covariates of a different length.

#### Scenario: A fractional duration
- **WHEN** a duration is 2.5
- **THEN** an error names its row index
