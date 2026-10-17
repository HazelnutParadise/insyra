## ADDED Requirements

### Requirement: A failed creation leaves no environment

`Manager.Create` SHALL remove the environment directory it made when writing the default files fails, and SHALL return that error. A default file path that holds anything other than a regular file SHALL be an error naming the file, not skipped.

#### Scenario: A directory where a default file belongs
- **WHEN** the path of `history.txt` holds a directory while the default files are written
- **THEN** the write fails with an error saying the path is not a regular file

#### Scenario: Creating twice
- **WHEN** an environment is created a second time
- **THEN** the call returns `ErrAlreadyExists` and the first environment is untouched
