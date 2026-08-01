## ADDED Requirements

### Requirement: A forest of decorrelated trees can be fitted
The system SHALL fit random forests for classification and regression: each tree grown on a bootstrap resample of the rows, each split restricted to a random subset of features, predictions aggregated across trees.

#### Scenario: A forest classifies

- **WHEN** a forest classifier predicts
- **THEN** it averages the trees' class probabilities and answers the largest
- **AND** the probabilities it reports are those averages, summing to one per row

#### Scenario: A forest regresses

- **WHEN** a forest regressor predicts
- **THEN** it answers the mean of its trees' predictions

#### Scenario: A bootstrap sample misses a class

- **WHEN** some tree's resample contains no observation of a class
- **THEN** that tree still scores probabilities over the full class list, in the shared order
- **AND** no column misalignment is possible between trees

#### Scenario: Feature importances are read

- **WHEN** a forest reports feature importances
- **THEN** they are the renormalized mean of the per-tree importances, one per feature, summing to one

### Requirement: A forest is reproducible from its seed
The system SHALL derive every random draw from one forest seed, report the seed on the fitted model, and produce identical forests from identical seeds regardless of parallel scheduling.

#### Scenario: The same seed is used twice

- **WHEN** two forests are fitted with the same seed on the same data
- **THEN** their predictions, probabilities and importances are identical

#### Scenario: No seed is supplied

- **WHEN** a forest is fitted without a seed
- **THEN** one is drawn and reported on the model
- **AND** refitting with the reported seed reproduces the forest exactly
