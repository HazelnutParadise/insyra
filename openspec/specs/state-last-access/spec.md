# state-last-access Specification

## Purpose
Keep environment access times typed in Go while preserving the timestamp wire format and variables shared with earlier releases.
## Requirements
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

### Requirement: Compatible output and preserved variables
State JSON MUST write lastAccess as an empty string for zero time and an RFC3339 string otherwise, readable by 0.3.6. Load/save and export/import MUST preserve variable values, including integers above 2^53 and unreadable variable representations.

#### Scenario: Cross-version state
- **WHEN** 0.4 reads a state file produced by the old writer and exports or saves its variables
- **THEN** all variables survive and the old reader can decode the output timestamp

#### Scenario: Fractional timestamp export
- **WHEN** an environment with a fractional RFC3339 timestamp is exported and imported
- **THEN** the timestamp retains the same instant and variables retain exact values

### Requirement: State carries no file format of its own
State MUST encode and decode with the standard JSON rules for its fields, so a struct that embeds it keeps its own fields. The state.json layout MUST be described by one unexported type used by every reader and writer of the file and of an export.

#### Scenario: Embedding State
- **WHEN** a caller's struct embeds State beside its own field and is encoded and decoded with encoding/json
- **THEN** its own field and LastAccess survive

