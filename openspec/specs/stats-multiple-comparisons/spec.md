# stats-multiple-comparisons Specification

## Purpose
Correcting a batch of p-values for testing many hypotheses at once, matching R p.adjust, so that a study running dozens of tests reports error rates that account for all of them.

## Requirements

### Requirement: A batch of p-values is corrected together
`stats.AdjustPValues` SHALL return the p-values adjusted by Holm, Bonferroni, Hochberg, Benjamini-Hochberg or Benjamini-Yekutieli in the order given, each at most 1, equal to R `p.adjust` for the same method, and SHALL leave a NaN as NaN without counting it among the tests.

#### Scenario: Holm matches R
- **WHEN** the p-values 0.01, 0.04, 0.03, 0.005 are adjusted with Holm
- **THEN** the result is 0.03, 0.06, 0.06, 0.02

#### Scenario: A test that was not run
- **WHEN** a list holds two NaN among four p-values
- **THEN** the NaN positions stay NaN and the number of tests is two

### Requirement: Values that are not p-values are refused
`stats.AdjustPValues` SHALL return an error for a nil or empty list, a value that is not a number, an infinity, a value below 0 or above 1, and an unknown method.

#### Scenario: A value above one
- **WHEN** the list holds 1.5
- **THEN** an error names the row index of that value
