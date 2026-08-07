## ADDED Requirements

### Requirement: Per-class classification quality is measurable
The system SHALL score class predictions with precision, recall and F1, combined across classes by a caller-chosen averaging mode, and SHALL declare that a larger score is better for all three.

#### Scenario: No averaging mode is chosen

- **WHEN** a caller scores predictions without choosing an averaging mode
- **THEN** the unweighted mean over every observed class is returned
- **AND** this works for any number of classes without naming one

#### Scenario: Micro and weighted averaging are requested

- **WHEN** a caller chooses micro or support-weighted averaging
- **THEN** micro combines the per-class counts before dividing
- **AND** weighted combines the per-class scores in proportion to how often each class actually occurs

#### Scenario: One class is scored

- **WHEN** a caller chooses binary averaging and names the positive class
- **THEN** precision, recall and F1 are computed for that class alone

#### Scenario: Binary averaging without a named positive class

- **WHEN** a caller chooses binary averaging without naming the positive class
- **THEN** the request is refused with an error
- **AND** no class is chosen on the caller's behalf, because the score is not invariant under that choice

#### Scenario: A positive class is named under a non-binary average

- **WHEN** a positive class is named but the averaging mode does not use one
- **THEN** the request is refused rather than the name being silently ignored

#### Scenario: A class is never predicted

- **WHEN** some class occurs in the true labels but never in the predictions
- **THEN** its precision contributes zero rather than failing the whole evaluation
- **AND** the convention is documented

### Requirement: The per-class metrics agree with the reference implementation
The system SHALL produce the same precision, recall and F1 as scikit-learn's reference computation for every averaging mode, verified where the reference is installed.

#### Scenario: The reference toolchain is present

- **WHEN** the comparison runs on a machine with scikit-learn available
- **THEN** macro, micro, weighted and binary results match the reference

#### Scenario: The reference toolchain is absent

- **WHEN** scikit-learn is not available
- **THEN** the comparison reports through the shared reference-toolchain gate rather than skipping on its own
