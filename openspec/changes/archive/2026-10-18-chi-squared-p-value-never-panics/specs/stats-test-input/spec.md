## ADDED Requirements

### Requirement: Chi-squared based tests return a result on degenerate input
The system SHALL return a result, never panic, from a chi-squared based test when its statistic is zero or rounds to a tiny negative value, reporting a p-value of 1, and SHALL report a NaN p-value when the degrees of freedom are not positive.

#### Scenario: All observations equal
- **WHEN** `KruskalWallis` or `FriedmanTest` receives lists whose values are all equal
- **THEN** it returns a result with a p-value of 1 and no panic

#### Scenario: Identical groups
- **WHEN** `BartlettTest` receives two identical groups
- **THEN** it returns a result with a p-value of 1 and no panic
