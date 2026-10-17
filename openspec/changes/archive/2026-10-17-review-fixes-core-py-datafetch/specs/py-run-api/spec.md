## ADDED Requirements

### Requirement: Result field names match by strings.EqualFold

When a key in a result matches no struct field exactly, the field that takes it SHALL be the first field, in declaration order, whose name `strings.EqualFold` accepts for the key. A key that folds to a field's name under any other rule MUST NOT match that field.

#### Scenario: A dotted capital I does not match I

- **WHEN** the result holds a key `İ` and the struct has a field `I` with no exact match
- **THEN** the field `I` keeps its zero value
