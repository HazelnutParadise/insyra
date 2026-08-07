## ADDED Requirements

### Requirement: Candidate estimators are compared on identical folds
The system SHALL evaluate every candidate in a grid search on the same fold assignment, and SHALL make the assignment reproducible.

#### Scenario: No seed is supplied

- **WHEN** a grid search runs without a seeded sampling option
- **THEN** one seed is drawn, applied to every candidate
- **AND** the seed is reported on the result so the run can be repeated

#### Scenario: Two identical candidates are searched

- **WHEN** two candidates with identical fitting behaviour are compared
- **THEN** their per-fold scores are identical, because their folds were

### Requirement: The winner is chosen by the metric's direction and refitted
The system SHALL rank candidates by the metric's declared direction, keep the earliest candidate on ties, and return the winner refitted on the full training data.

#### Scenario: A loss metric ranks the grid

- **WHEN** candidates are scored by a metric that improves downward
- **THEN** the candidate with the smallest mean wins

#### Scenario: The winner is used immediately

- **WHEN** a grid search completes
- **THEN** the returned model is the winning estimator fitted on all supplied rows, not on a fold subset

#### Scenario: A directionless metric is supplied

- **WHEN** the metric declares no direction
- **THEN** the search is refused before any fitting

#### Scenario: Candidates cannot be told apart

- **WHEN** a candidate has no name, or two candidates share one
- **THEN** the search is refused, because a score nobody can attribute to a configuration is unusable

#### Scenario: A candidate fails to fit

- **WHEN** any candidate's fit returns an error on any fold
- **THEN** the search fails naming that candidate
