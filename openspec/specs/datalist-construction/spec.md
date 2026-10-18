# datalist-construction Specification

## Purpose
Defines how NewDataList builds a list from the values it is given, including how deeply it flattens nested slices, so construction always ends with a usable list instead of crashing.

## Requirements
### Requirement: NewDataList bounds how deep it flattens
The system SHALL flatten nested slices passed to `NewDataList` down to 64 levels and SHALL keep a slice found below that depth as a single cell, so that input containing a slice that refers to itself returns a list instead of ending the process.

#### Scenario: A slice that contains itself
- **WHEN** `NewDataList` receives a slice whose element is the slice itself
- **THEN** it returns a list and the program keeps running

#### Scenario: Ordinary nesting
- **WHEN** `NewDataList` receives slices nested a few levels deep
- **THEN** every leaf value becomes one cell, as before

### Requirement: Shift preserves cell boundaries
The system SHALL return from `DataList.Shift` a list with the same number of cells as the input, in which a cell that held a slice still holds that slice as one value.

#### Scenario: A slice cell survives a shift
- **WHEN** a list holding `[]byte{1, 2}` and `3` is shifted by one period
- **THEN** the result has two cells: nil, then the `[]byte{1, 2}`

