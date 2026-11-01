# stats-discrete-time-hazard Specification

## Purpose
How `stats.DiscreteTimeHazard` estimates whether a spell is more or less likely to end the longer it lasts, by a person-period binomial GLM that matches R's glm.
## Requirements
### Requirement: The hazard model is the person-period binomial GLM

`stats.DiscreteTimeHazard` SHALL fit a binomial GLM of the person-period event indicator on the baseline terms and the covariates, with the logit or cloglog link, and its coefficients, standard errors and deviance SHALL equal R's `glm` on the same table. It SHALL return the terms in the order Intercept, baseline, covariates, and `Duration` SHALL be the `LogPeriod` or `Period` term, nil for one level per period.

#### Scenario: Log period and a covariate

- **WHEN** the 60 spells of `testdata/discrete_hazard.R` are fitted with the covariate x
- **THEN** the coefficients and standard errors equal R's within 1e-6 relative under both links

### Requirement: One level per period reproduces the life table

With `HazardBaselineDummies` and no covariates, the fitted hazard of each period SHALL equal the events over the spells at risk in that period, and a period in which no spell or every spell at risk ended SHALL be an error naming it.

#### Scenario: Thirteen spells over three periods

- **WHEN** the model is fitted with one level per period
- **THEN** each period's hazard equals KaplanMeier's Events over AtRisk

### Requirement: Spells are whole periods

`DiscreteTimeHazard` and `PersonPeriod` SHALL refuse a duration that is not a whole number of at least 1 and lists of different lengths, and `DiscreteTimeHazard` SHALL refuse spells none of which ended, a link other than Logit or Cloglog, and more than one options value.

#### Scenario: A zero duration

- **WHEN** a duration is 0
- **THEN** an error names its row index

### Requirement: The cloglog link matches R

`GLM` SHALL accept `Cloglog` for the binomial family with `eta = log(-log(1 - mu))`, as R's `binomial("cloglog")`, and SHALL keep refusing `Probit`.

#### Scenario: A cloglog fit

- **WHEN** the person-period table is fitted with the cloglog link
- **THEN** the deviance equals R's within 1e-8 relative

