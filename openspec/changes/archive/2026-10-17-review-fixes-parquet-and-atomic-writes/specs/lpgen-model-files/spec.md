## MODIFIED Requirements

### Requirement: Saving a model replaces the file only when it is whole

`GenerateLPFile` SHALL write through a temporary file in the directory of the file `filename` names (the target, when `filename` is a symbolic link) that is renamed into place only after the whole model was written, so a failed call SHALL NOT leave a file at `filename` or a temporary file behind, and a file already at `filename` SHALL keep its content. A successful call SHALL save exactly the text `WriteLP` writes.

#### Scenario: An existing file and a failed save
- **WHEN** `path` already holds an LP file, and `GenerateLPFile(path)` is called on a model with an unknown objective type
- **THEN** the call returns an error and the content of `path` is the same as before the call

#### Scenario: A successful save
- **WHEN** a valid model is saved with `GenerateLPFile(path)`
- **THEN** the call returns nil and the file content equals the text `WriteLP` writes
