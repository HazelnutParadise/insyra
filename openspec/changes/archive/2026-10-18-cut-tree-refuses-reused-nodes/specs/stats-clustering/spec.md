## ADDED Requirements

### Requirement: Tree cutting refuses a node joined more than once
The system SHALL return an error from CutTreeByK and CutTreeByHeight when any leaf or any merge of the tree is joined by more than one merge step, naming the step and the node.

#### Scenario: A leaf is joined twice
- **WHEN** two merge steps both join leaf -1
- **THEN** cutting the tree returns an error
