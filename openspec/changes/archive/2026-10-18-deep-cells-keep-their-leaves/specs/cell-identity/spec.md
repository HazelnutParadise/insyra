## ADDED Requirements

### Requirement: A value past the depth limit is not merged with a different one
Past the encoder's nesting limit, an interface SHALL be looked through: a scalar it holds SHALL be identified by kind and value, and a container by its address, so that two deeply nested values that differ in a leaf SHALL NOT count as one.

#### Scenario: Two deep values with different leaves
- **WHEN** a list holds a value nested 40 `[]any` deep with leaf 1 and another with leaf 2
- **THEN** counting the first finds one occurrence, not two
