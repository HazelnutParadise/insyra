## ADDED Requirements

### Requirement: Creation and modification times keep the clock's resolution

`DataList` and `DataTable` SHALL provide `CreationTime() time.Time` and `LastModifiedTime() time.Time`. Both SHALL keep the clock's full resolution, so two changes less than a second apart give different `LastModifiedTime` values whenever the clock reads differently at each.

#### Scenario: Two changes within one second
- **WHEN** the clock reads T when a list is created, T+100ns at the first `Append` and T+200ns at the second
- **THEN** `CreationTime()` is T, and `LastModifiedTime()` is T+100ns after the first `Append` and T+200ns after the second

#### Scenario: A table
- **WHEN** the clock reads T when a table is created and T+5ns when a row is appended
- **THEN** `CreationTime()` is T and `LastModifiedTime()` is T+5ns

### Requirement: The modification time never moves backwards

`LastModifiedTime` SHALL NOT decrease when a change happens while the wall clock reads earlier than the recorded time.

#### Scenario: The clock steps back
- **WHEN** a list is modified at T+1s and then again after the clock returns to T
- **THEN** `LastModifiedTime()` is still T+1s

### Requirement: The second-resolution getters remain until 0.4

`GetCreationTimestamp()` and `GetLastModifiedTimestamp()` SHALL be marked Deprecated and SHALL return the Unix seconds of `CreationTime()` and `LastModifiedTime()`.

#### Scenario: Old getters
- **WHEN** a list is created at `time.Unix(1700000000, 500)`
- **THEN** `GetCreationTimestamp()` is 1700000000 and `CreationTime()` equals `time.Unix(1700000000, 500)`

### Requirement: IsTheSameAs compares times in whole seconds

`DataList.IsTheSameAs` SHALL keep comparing the creation and modification times of the two lists in whole Unix seconds, in addition to the data and the name, so storing the times at full resolution does not change its answer.

#### Scenario: A clone made in the same second
- **WHEN** a list is cloned 500ms after it was created
- **THEN** `IsTheSameAs(clone)` is true

#### Scenario: A clone made in a later second
- **WHEN** a list is cloned two seconds after it was created
- **THEN** `IsTheSameAs(clone)` is false
