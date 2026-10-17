## ADDED Requirements

### Requirement: ReadJSON reads a Marshaler before a Reader

`ReadJSON` SHALL read a value that implements both `json.Marshaler` and `io.Reader` as the JSON its `MarshalJSON` returns, and SHALL NOT read it as a stream.

#### Scenario: A value that is both
- **WHEN** a value's `MarshalJSON` returns `{"a": 17}` and its reader holds `{"a": 3}`, and the value is passed to `ReadJSON`
- **THEN** the table's column `a` holds 17
