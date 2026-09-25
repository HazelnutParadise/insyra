## ADDED Requirements

### Requirement: Training batch normalization takes its settings by name

`Tape.BatchNormalizationTraining` and `BatchNormTraining` SHALL take momentum and epsilon as fields of one optional `BatchNormOptions`, a zero field keeping torch's default, and SHALL refuse more than one.

#### Scenario: Only epsilon set

- **WHEN** training batch normalization runs with `BatchNormOptions{Epsilon: 1e-3}`
- **THEN** momentum keeps its default of 0.1 and epsilon is 1e-3
