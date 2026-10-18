## ADDED Requirements

### Requirement: Nested strings are written unambiguously
The system SHALL write each string inside a nested value with its length, so that two nested values count as the same value only when their strings are equal element by element.

#### Scenario: A string holding separators
- **WHEN** a list holds Cell([]any{"a,s:b"}) and Cell([]any{"a","b"})
- **THEN** Counter reports two entries and Count of each finds one
