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

