## ADDED Requirements

### Requirement: The survival curve matches the Kaplan-Meier estimator
`stats.KaplanMeier` SHALL return, for each distinct event time, the number at risk, events and censored, the survival probability, Greenwood's standard error of it, and the median, with a subject censored at an event time counted as at risk for that event.

#### Scenario: Censored data against R
- **WHEN** 16 subjects with 5 censored are analysed
- **THEN** the survival values, standard errors and median 9 equal R `survfit`

#### Scenario: A curve that reaches exactly one half
- **WHEN** four events occur at 1, 2, 3 and 4
- **THEN** the median is 2.5

### Requirement: Unusable input is refused
`stats.KaplanMeier` SHALL return an error for a nil or empty list, lists of different lengths, a negative or non-finite duration and an event flag that is not a bool, 0 or 1.

#### Scenario: A flag of 2
- **WHEN** an observed value is 2
- **THEN** an error names its row index
