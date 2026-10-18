## ADDED Requirements

### Requirement: The deprecated GLPK functions never return a nil table
`lp.SolveFromFile` and `lp.SolveModel` SHALL return a non-nil solution table and a non-nil info table on every path. When no solution is produced, the solution table SHALL be empty and its `Err()` SHALL hold the reason.

#### Scenario: Rejected arguments
- **WHEN** `SolveFromFile` is called with more than one timeout
- **THEN** both tables are non-nil, the solution table is empty with a non-nil `Err()`, and the info table's Status is `Error`

#### Scenario: Unreadable solution file
- **WHEN** the solution file glpsol should have written is missing
- **THEN** the parsed table is empty and its `Err()` holds the reason
