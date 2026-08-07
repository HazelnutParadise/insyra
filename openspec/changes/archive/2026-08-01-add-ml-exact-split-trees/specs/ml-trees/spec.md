## ADDED Requirements

### Requirement: Numeric splits can be exact
The system SHALL offer an exact split search that considers the midpoint between every pair of adjacent distinct values of each numeric feature, alongside the default histogram search.

#### Scenario: Exact splitting is requested

- **WHEN** a tree is fitted with exact splits
- **THEN** every midpoint between adjacent distinct numeric values is a split candidate
- **AND** categorical features, missing-value routing and growth bounds behave as they do under the histogram search

#### Scenario: Exact splitting is combined with a bin cap

- **WHEN** exact splits are requested together with an explicit bin cap
- **THEN** the fit is refused, because a capped exact search is neither of the two things it names

#### Scenario: An ensemble requests exact splits

- **WHEN** a forest or boosted ensemble passes exact splits through its tree options
- **THEN** every tree in the ensemble splits exactly

### Requirement: The exact tree matches the reference implementation's predictions
The system SHALL produce, for the same data and depth, the same predictions as scikit-learn's decision tree — prediction for prediction, not merely the same accuracy — verified where the reference is installed.

#### Scenario: The reference is present

- **WHEN** the comparison runs on a machine with scikit-learn available
- **THEN** an exact classification tree's predicted labels match scikit-learn's on every probe point
- **AND** an exact regression tree's predictions match within single-precision tolerance
