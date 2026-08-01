## ADDED Requirements

### Requirement: The positive class of a ranking metric can be named
**WITHDRAWN — never merged.** Kept as the record of what was proposed. The
premise was that the default positive class could report the complement of the
intended score; measurement showed the area under the ROC curve is invariant
under the swap, because this metric receives the whole probability table and
swapping the class swaps the score column with it. See `proposal.md`.

The system SHALL let a caller name which class a binary ranking metric treats as positive, and SHALL state which class it uses when none is named.

#### Scenario: A caller names the positive class

- **WHEN** a caller scores a binary classifier with a ranking metric and names the positive class
- **THEN** the score is computed against that class

#### Scenario: No positive class is named

- **WHEN** no positive class is named
- **THEN** the second of the model's classes is used
- **AND** this default is stated in the documentation alongside the fact that class order comes from the sorted training labels

#### Scenario: The named class is not one the model knows

- **WHEN** the named positive class is not among the model's classes
- **THEN** the request is refused with an error listing the classes the model does have
