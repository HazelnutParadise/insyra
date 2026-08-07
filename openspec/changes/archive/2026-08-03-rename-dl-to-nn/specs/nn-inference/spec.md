## ADDED Requirements

### Requirement: The neural-network package is importable as nn

The deep-learning package SHALL be importable as
`github.com/HazelnutParadise/insyra/nn`, and no `dl` import path SHALL
remain in the repository. Documentation examples SHALL not collide with
the `dl`/`DL` DataList idiom.

#### Scenario: The old import path is gone

- **WHEN** the repository is searched for the old import path
- **THEN** no Go file SHALL import `insyra/dl`, and the full test suite
  SHALL pass under the new name
