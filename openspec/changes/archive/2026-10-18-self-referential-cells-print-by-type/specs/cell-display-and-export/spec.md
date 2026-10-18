## ADDED Requirements

### Requirement: A cell that refers to itself is shown by its type
`Show` and the error messages of `stats` SHALL name a cell by its Go type when printing its contents would not finish, so that a slice containing itself SHALL NOT end the process.

#### Scenario: Showing a self-referential slice
- **WHEN** a list holds a slice whose element is the slice itself and `Show` is called
- **THEN** the cell is shown as its type and the call returns
