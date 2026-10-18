## ADDED Requirements

### Requirement: A malformed buffer is refused with an error
The system SHALL return an error that names the column when `ExecuteNearestExact` receives a buffer declared numeric whose `Values` it cannot read, whose `Type` disagrees with its `Values`, or whose length differs from another column's, and SHALL NOT return an empty result with a nil error for it.

#### Scenario: Values of an unreadable type
- **WHEN** a buffer with `Type` `float64` holds `[]float32{1, 2}`
- **THEN** the call returns an error naming the column
- **AND** no empty result is reported as success

#### Scenario: Type disagrees with Values
- **WHEN** a buffer with `Type` `int64` holds `[]float64`
- **THEN** the call returns an error naming the column

#### Scenario: Columns of different lengths
- **WHEN** two buffers hold different numbers of values
- **THEN** the call returns an error naming the shorter or longer column

#### Scenario: A non-numeric column stays ineligible
- **WHEN** a buffer has `Type` `string` and holds `[]string`
- **THEN** the call returns no answer with reason `dtype-not-eligible` and a nil error outside strict mode
