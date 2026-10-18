## ADDED Requirements

### Requirement: Shift preserves cell boundaries
The system SHALL return from `DataList.Shift` a list with the same number of cells as the input, in which a cell that held a slice still holds that slice as one value.

#### Scenario: A slice cell survives a shift
- **WHEN** a list holding `[]byte{1, 2}` and `3` is shifted by one period
- **THEN** the result has two cells: nil, then the `[]byte{1, 2}`
