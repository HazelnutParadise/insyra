# ml-trees Specification

## Purpose
TBD - created by archiving change add-ml-decision-tree. Update Purpose after archive.
## Requirements
### Requirement: Trees for classification and regression
The system SHALL fit a decision tree from labelled observations and score new ones with it.

#### Scenario: A classification tree is fitted and scores new data
- **WHEN** a caller fits a classification tree on labelled observations
- **THEN** it predicts a label for each new observation
- **AND** it reports class probabilities whose columns match its classes in order

#### Scenario: A regression tree is fitted and scores new data
- **WHEN** a caller fits a regression tree on continuous targets
- **THEN** it predicts a continuous value for each new observation

#### Scenario: Growth is bounded
- **WHEN** a caller bounds the tree's depth, its leaf count, or the observations a leaf must hold
- **THEN** the fitted tree respects every bound given

### Requirement: The same inputs produce the same tree
The system SHALL fit deterministically.

#### Scenario: The same fit is repeated
- **WHEN** the same observations and options are fitted twice
- **THEN** the two trees are identical, including the order of tied splits

#### Scenario: Two candidate splits score equally
- **WHEN** two splits have exactly equal gain
- **THEN** the choice between them is resolved by a stated rule rather than by whichever was evaluated first

### Requirement: Missing values are learned, not imputed
The system SHALL route missing values by a direction learned during fitting.

#### Scenario: A feature is missing at fitting time
- **WHEN** observations carry missing values in a feature used for a split
- **THEN** the split records which branch missing values take
- **AND** that direction is the one that scored better during fitting

#### Scenario: A feature is missing only at scoring time
- **WHEN** a value is missing in a feature whose split saw none during fitting
- **THEN** it is routed by a stated default rather than producing an error or an arbitrary answer

### Requirement: Categorical features are split without encoding
The system SHALL split a categorical feature by a subset of its categories.

#### Scenario: A categorical feature is used
- **WHEN** a feature is declared categorical
- **THEN** the split divides its categories into two groups rather than thresholding an imposed numeric order

#### Scenario: An unseen category appears at scoring time
- **WHEN** a category not present during fitting appears
- **THEN** it is routed by a stated default rather than producing an error

### Requirement: Precision follows the contract
The system SHALL hold each quantity at the precision its role requires.

#### Scenario: Features are binned and accumulated
- **WHEN** a tree is fitted
- **THEN** feature values are held at single precision or narrower
- **AND** the quantities accumulated across observations are held so that the result does not depend on the order they were summed

#### Scenario: A result is reported
- **WHEN** a leaf value or an importance is reported to a caller
- **THEN** it is reported in double precision

### Requirement: Feature importances are reported
The system SHALL report each feature's contribution.

#### Scenario: Importances are requested
- **WHEN** a caller asks a fitted tree for feature importances
- **THEN** one value is returned per fitted feature, in the order the model reports its features

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

### Requirement: Boosted tree ensembles can be fitted
The system SHALL fit gradient-boosted tree ensembles: regression under squared loss by residual fitting, and binary classification under logistic loss with second-order leaf values.

#### Scenario: More stages fit the training data no worse

- **WHEN** two boosted regressors differ only in stage count
- **THEN** the one with more stages fits the training data at least as well

#### Scenario: A boosted classifier reports probabilities

- **WHEN** a boosted binary classifier reports probabilities
- **THEN** each row's two probabilities sum to one
- **AND** the predicted label is the class whose probability exceeds one half

#### Scenario: The residuals run out early

- **WHEN** the residuals reach zero before the requested stage count
- **THEN** fitting stops and the model reports how many stages ran

#### Scenario: A multiclass target is supplied

- **WHEN** the classification target holds more than two classes
- **THEN** the fit is refused with an error naming the binary limit

### Requirement: Boosting is deterministic
The system SHALL produce identical boosted ensembles from identical data and options, with no randomness involved.

#### Scenario: The same fit runs twice

- **WHEN** the same data and options are fitted twice
- **THEN** the predictions are identical, with no seed to manage

