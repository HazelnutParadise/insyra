## ADDED Requirements

### Requirement: NewDataList bounds how deep it flattens
The system SHALL flatten nested slices passed to `NewDataList` down to 64 levels and SHALL keep a slice found below that depth as a single cell, so that input containing a slice that refers to itself returns a list instead of ending the process.

#### Scenario: A slice that contains itself
- **WHEN** `NewDataList` receives a slice whose element is the slice itself
- **THEN** it returns a list and the program keeps running

#### Scenario: Ordinary nesting
- **WHEN** `NewDataList` receives slices nested a few levels deep
- **THEN** every leaf value becomes one cell, as before
