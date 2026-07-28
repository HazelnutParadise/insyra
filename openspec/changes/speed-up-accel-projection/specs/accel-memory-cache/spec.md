## ADDED Requirements

### Requirement: Projection does not allocate per element
The system SHALL project a typed column without allocating memory proportional to the number of values beyond the output buffers themselves.

#### Scenario: A large numeric column is projected
- **WHEN** a numeric column of N values is projected
- **THEN** the number of allocations does not grow with N
- **AND** a benchmark records the allocation count so a regression is visible

### Requirement: Projection accepts the same values it accepts today
The system SHALL continue to project every value it currently projects, including values whose named type has a numeric underlying kind.

#### Scenario: A column holds a named numeric type
- **WHEN** a column holds values of a type declared as a named numeric type, such as `type Celsius float64`
- **THEN** the column projects to the same buffer type and values as before

#### Scenario: A column holds a type with no numeric representation
- **WHEN** a column holds a value that cannot be represented as a number, and the column is otherwise numeric
- **THEN** projection reports the same error it reports today, naming the offending index
