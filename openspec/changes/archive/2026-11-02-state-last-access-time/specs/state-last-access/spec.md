## ADDED Requirements

### Requirement: Typed access time
State.LastAccess MUST be time.Time on 0.4, and EnvironmentInfo MUST receive that value without parsing text in the manager.

#### Scenario: Load a dated environment
- **WHEN** a state file contains an RFC3339 timestamp
- **THEN** LoadState and Info report the same instant and retain all variables

### Requirement: Compatible timestamp input
The state reader MUST accept empty, missing, null and invalid timestamp text as zero time without losing variables. Valid RFC3339 timestamps MUST retain their instant, including fractional seconds and offsets. Malformed documents and non-string non-null timestamps MUST retain their existing decode errors.

#### Scenario: Older empty timestamp
- **WHEN** an old state file contains lastAccess as an empty string
- **THEN** its variables restore and LastAccess is zero

#### Scenario: Invalid text is only unknown metadata
- **WHEN** lastAccess contains text that cannot be parsed
- **THEN** variables restore and LastAccess is zero

#### Scenario: Invalid JSON field type
- **WHEN** lastAccess is a number, boolean, array or object
- **THEN** state loading returns an error without writing the file

### Requirement: Compatible output and preserved variables
State JSON MUST write lastAccess as an empty string for zero time and an RFC3339 string otherwise, readable by 0.3.6. Load/save and export/import MUST preserve variable values, including integers above 2^53 and unreadable variable representations.

#### Scenario: Cross-version state
- **WHEN** 0.4 reads a state file produced by the old writer and exports or saves its variables
- **THEN** all variables survive and the old reader can decode the output timestamp

#### Scenario: Fractional timestamp export
- **WHEN** an environment with a fractional RFC3339 timestamp is exported and imported
- **THEN** the timestamp retains the same instant and variables retain exact values
