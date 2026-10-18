## ADDED Requirements

### Requirement: A scaler read from JSON refuses an empty column reference
The system SHALL refuse scaler JSON in which a fitted column has an empty reference, return an error naming the empty reference, and leave the receiver unchanged.

#### Scenario: Empty reference
- **WHEN** scaler JSON holds a column whose ref is an empty string
- **THEN** reading it returns an error and nothing is fitted
