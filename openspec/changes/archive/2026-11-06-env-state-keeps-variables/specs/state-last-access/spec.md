## MODIFIED Requirements

### Requirement: Typed access time
State.LastAccess MUST be a time.Time whose zero value means the last access is unknown, and EnvironmentInfo MUST receive that value without parsing text in the manager.

#### Scenario: Load a dated environment
- **WHEN** a state file contains an RFC3339 timestamp
- **THEN** LoadState and Info report the same instant and retain all variables

### Requirement: Compatible timestamp input
The state reader MUST load a lastAccess that is not RFC3339 text JSON can write back, including an empty, missing or null value and a value of another JSON type, as zero time without losing variables. Valid RFC3339 timestamps MUST retain their instant, including fractional seconds and offsets.

#### Scenario: Older empty timestamp
- **WHEN** an old state file contains lastAccess as an empty string
- **THEN** its variables restore and LastAccess is zero

#### Scenario: Invalid text is only unknown metadata
- **WHEN** lastAccess contains text that cannot be parsed
- **THEN** variables restore and LastAccess is zero

#### Scenario: Invalid JSON field type
- **WHEN** lastAccess is a number, boolean, array or object
- **THEN** variables restore, LastAccess is zero, and the next save keeps the variables

## ADDED Requirements

### Requirement: State carries no file format of its own
State MUST encode and decode with the standard JSON rules for its fields, so a struct that embeds it keeps its own fields. The state.json layout MUST be described by one unexported type used by every reader and writer of the file and of an export.

#### Scenario: Embedding State
- **WHEN** a caller's struct embeds State beside its own field and is encoded and decoded with encoding/json
- **THEN** its own field and LastAccess survive
