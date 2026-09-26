## ADDED Requirements

### Requirement: Err answers on a nil receiver

`Err()` on a nil `*DataList` or `*DataTable` SHALL return a new error naming the nil type instead of crashing. No other method SHALL be made safe on a nil receiver by this requirement.

#### Scenario: A lookup that found nothing

- **WHEN** `dt.GetCol(Name("missing"))` returns nil and `Err()` is called on the result
- **THEN** it returns an error saying `nil DataList`
